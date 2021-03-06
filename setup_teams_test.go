// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSetupTeams(t *testing.T) {
	clearDb()
	defer clearDb()
	var err error
	db, err = OpenDatabase(testDbPath)
	assert.Nil(t, err)
	defer db.Close()
	eventSettings, _ = db.GetEventSettings()

	// Check that there are no teams to start.
	recorder := getHttpResponse("/setup/teams")
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "0 teams")

	// Mock the URL to download team info from.
	teamInfoBody := "<PRE>\nID_team\tteam_number\tteam_name\tteam_name_short\tteam_city\tteam_stateprov\t" +
		"team_country\tteam_nickname team_rookieyear robot_name\n1\t254\tNASA\tChezy\tThe Cheesy Poofs\t" +
		"San Jose\tCA\tUSA\t1999\tBarrage\n</PRE>"
	teamInfoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, teamInfoBody)
	}))
	defer teamInfoServer.Close()
	officialTeamInfoUrl = teamInfoServer.URL

	// Add some teams.
	recorder = postHttpResponse("/setup/teams", "teamNumbers=254\r\nnotateam\r\n1114\r\n")
	assert.Equal(t, 302, recorder.Code)
	recorder = getHttpResponse("/setup/teams")
	assert.Contains(t, recorder.Body.String(), "2 teams")
	assert.Contains(t, recorder.Body.String(), "The Cheesy Poofs")
	assert.Contains(t, recorder.Body.String(), "1114")

	// Add another team.
	recorder = postHttpResponse("/setup/teams", "teamNumbers=33")
	assert.Equal(t, 302, recorder.Code)
	recorder = getHttpResponse("/setup/teams")
	assert.Contains(t, recorder.Body.String(), "3 teams")
	assert.Contains(t, recorder.Body.String(), "33")

	// Edit a team.
	recorder = getHttpResponse("/setup/teams/254/edit")
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "The Cheesy Poofs")
	recorder = postHttpResponse("/setup/teams/254/edit", "nickname=Teh Chezy Pofs")
	assert.Equal(t, 302, recorder.Code)
	recorder = getHttpResponse("/setup/teams")
	assert.Contains(t, recorder.Body.String(), "Teh Chezy Pofs")

	// Delete a team.
	recorder = postHttpResponse("/setup/teams/1114/delete", "")
	assert.Equal(t, 302, recorder.Code)
	recorder = getHttpResponse("/setup/teams")
	assert.Contains(t, recorder.Body.String(), "2 teams")

	// Test clearing all teams.
	recorder = postHttpResponse("/setup/teams/clear", "")
	assert.Equal(t, 302, recorder.Code)
	recorder = getHttpResponse("/setup/teams")
	assert.Contains(t, recorder.Body.String(), "0 teams")
}

func TestSetupTeamsDisallowModification(t *testing.T) {
	clearDb()
	defer clearDb()
	var err error
	db, err = OpenDatabase(testDbPath)
	assert.Nil(t, err)
	defer db.Close()
	eventSettings, _ = db.GetEventSettings()
	db.CreateTeam(&Team{Id: 254, Nickname: "The Cheesy Poofs"})
	db.CreateMatch(&Match{Type: "qualification"})

	// Disallow adding teams.
	recorder := postHttpResponse("/setup/teams", "teamNumbers=33")
	assert.Contains(t, recorder.Body.String(), "can't modify")
	assert.Contains(t, recorder.Body.String(), "1 teams")
	assert.Contains(t, recorder.Body.String(), "The Cheesy Poofs")

	// Disallow deleting team.
	recorder = postHttpResponse("/setup/teams/254/delete", "")
	assert.Contains(t, recorder.Body.String(), "can't modify")
	assert.Contains(t, recorder.Body.String(), "1 teams")
	assert.Contains(t, recorder.Body.String(), "The Cheesy Poofs")

	// Disallow clearing all teams.
	recorder = postHttpResponse("/setup/teams/clear", "")
	assert.Contains(t, recorder.Body.String(), "can't modify")
	assert.Contains(t, recorder.Body.String(), "1 teams")
	assert.Contains(t, recorder.Body.String(), "The Cheesy Poofs")

	// Allow editing a team.
	recorder = postHttpResponse("/setup/teams/254/edit", "nickname=Teh Chezy Pofs")
	assert.Equal(t, 302, recorder.Code)
	recorder = getHttpResponse("/setup/teams")
	assert.NotContains(t, recorder.Body.String(), "can't modify")
	assert.Contains(t, recorder.Body.String(), "1 teams")
	assert.Contains(t, recorder.Body.String(), "Teh Chezy Pofs")
}

func TestSetupTeamsBadReqest(t *testing.T) {
	clearDb()
	defer clearDb()
	var err error
	db, err = OpenDatabase(testDbPath)
	assert.Nil(t, err)
	defer db.Close()

	recorder := getHttpResponse("/setup/teams/254/edit")
	assert.Equal(t, 400, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "No such team")
	recorder = postHttpResponse("/setup/teams/254/edit", "")
	assert.Equal(t, 400, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "No such team")
	recorder = postHttpResponse("/setup/teams/254/delete", "")
	assert.Equal(t, 400, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "No such team")
}

func TestSetupTeamsWpaKeys(t *testing.T) {
	clearDb()
	defer clearDb()
	var err error
	db, err = OpenDatabase(testDbPath)
	assert.Nil(t, err)
	defer db.Close()
	eventSettings, _ = db.GetEventSettings()
	eventSettings.NetworkSecurityEnabled = true

	team1 := &Team{Id: 254, WpaKey: "aaaaaaaa"}
	team2 := &Team{Id: 1114}
	db.CreateTeam(team1)
	db.CreateTeam(team2)

	recorder := getHttpResponse("/setup/teams/generate_wpa_keys?all=false")
	assert.Equal(t, 302, recorder.Code)
	team1, _ = db.GetTeamById(254)
	team2, _ = db.GetTeamById(1114)
	assert.Equal(t, "aaaaaaaa", team1.WpaKey)
	assert.Equal(t, 8, len(team2.WpaKey))

	recorder = getHttpResponse("/setup/teams/generate_wpa_keys?all=true")
	assert.Equal(t, 302, recorder.Code)
	team1, _ = db.GetTeamById(254)
	team3, _ := db.GetTeamById(1114)
	assert.NotEqual(t, "aaaaaaaa", team1.WpaKey)
	assert.Equal(t, 8, len(team1.WpaKey))
	assert.NotEqual(t, team2.WpaKey, team3.WpaKey)
	assert.Equal(t, 8, len(team3.WpaKey))

	// Disallow invalid manual WPA keys.
	recorder = postHttpResponse("/setup/teams/254/edit", "wpa_key=1234567")
	assert.Equal(t, 500, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "WPA key must be between 8 and 63 characters")
}

func TestSetupTeamsPublish(t *testing.T) {
	clearDb()
	defer clearDb()
	var err error
	db, err = OpenDatabase(testDbPath)
	assert.Nil(t, err)
	defer db.Close()
	eventSettings, _ = db.GetEventSettings()
	tbaBaseUrl = "fakeurl"
	eventSettings.TbaPublishingEnabled = true

	recorder := postHttpResponse("/setup/teams/publish", "")
	assert.Equal(t, 500, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Failed to publish teams")
}
