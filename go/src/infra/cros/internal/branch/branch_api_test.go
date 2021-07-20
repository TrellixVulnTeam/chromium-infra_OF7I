// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package branch

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andygrunwald/go-gerrit"
)

var (
	httpClient = &http.Client{}
	testMux    *http.ServeMux
	testClient *gerrit.Client
	testServer *httptest.Server
	largeQPS   = 1e9
)

func setUp() {
	testMux = http.NewServeMux()
	testServer = httptest.NewServer(testMux)
	testClient, _ = gerrit.NewClient(testServer.URL, nil)
}

func tearDown() {
	testServer.Close()
}

func testMethod(t *testing.T, r *http.Request, want string) {
	if got := r.Method; got != want {
		t.Errorf("Request method: %v, want %v", got, want)
	}
}

func TestCreateRemoteBranchesAPI_success(t *testing.T) {
	setUp()
	defer tearDown()

	branchesToCreate := []GerritProjectBranch{
		{GerritURL: testServer.URL, Project: "my-project", SrcRef: "my-source", Branch: "my-branch-1"},
		{GerritURL: testServer.URL, Project: "my-project", SrcRef: "my-source", Branch: "my-branch-2"},
	}

	branchesCreated := make(chan string, len(branchesToCreate))

	testMux.HandleFunc("/projects/my-project/branches/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")
		branchName := r.URL.Path[len("/projects/my-project/branches/"):]
		branchesCreated <- branchName

		defer r.Body.Close()
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, branchName, http.StatusBadRequest)
		}

		bi := &gerrit.BranchInput{}
		if err := json.Unmarshal(b, bi); err != nil {
			http.Error(w, branchName, http.StatusBadRequest)
		}
		info := &gerrit.BranchInfo{
			Ref: bi.Ref,
		}
		branchInfoRaw, err := json.Marshal(&info)
		if err != nil {
			http.Error(w, branchName, http.StatusBadRequest)
		}

		fmt.Fprint(w, `)]}'`+"\n"+string(branchInfoRaw))
	})

	c := &Client{}
	if err := c.CreateRemoteBranchesAPI(httpClient,
		branchesToCreate, false, largeQPS); err != nil {
		t.Error(err)
	}
	close(branchesCreated)
	branchMap := make(map[string]bool)
	for bc := range branchesCreated {
		branchMap[bc] = true
	}
	if len(branchesToCreate) != len(branchMap) {
		t.Errorf("expected %v branches created, instead %v", len(branchesToCreate), len(branchMap))
	}
	for _, btc := range branchesToCreate {
		if _, ok := branchMap[btc.Branch]; !ok {
			t.Errorf("no branch creation call made for %v", btc.Branch)
		}
	}
}

func TestCreateRemoteBranchesAPI_apiError(t *testing.T) {
	setUp()
	defer tearDown()

	branchesToCreate := []GerritProjectBranch{
		{GerritURL: testServer.URL, Project: "my-project", SrcRef: "my-source", Branch: "my-branch-1"},
	}

	testMux.HandleFunc("/projects/my-project/branches/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")
		branchName := r.URL.Path[len("/projects/my-project/branches/"):]
		http.Error(w, branchName, http.StatusBadRequest)
	})

	c := &Client{}
	if err := c.CreateRemoteBranchesAPI(httpClient,
		branchesToCreate, false, largeQPS); err != nil {
	} else {
		t.Errorf("expected an error, instead nil")
	}
}

func TestCheckSelfGroupMembership_success(t *testing.T) {
	setUp()
	defer tearDown()

	testMux.HandleFunc("/accounts/self/groups", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		info := []*gerrit.GroupInfo{
			{Name: "in this group"},
		}
		groupInfoRaw, err := json.Marshal(&info)
		if err != nil {
			http.Error(w, "groups", http.StatusBadRequest)
		}

		fmt.Fprint(w, `)]}'`+"\n"+string(groupInfoRaw))
	})

	inGroup, err := CheckSelfGroupMembership(httpClient, testServer.URL, "in this group")
	if err != nil {
		t.Error(err)
	}
	if !inGroup {
		t.Errorf("expected to be in group, but wasn't")
	}

	inGroup, err = CheckSelfGroupMembership(httpClient, testServer.URL, "not in this group")
	if err != nil {
		t.Error(err)
	}
	if inGroup {
		t.Errorf("expected not to be in group, but was")
	}
}

func TestCheckSelfGroupMembership_apiError(t *testing.T) {
	setUp()
	defer tearDown()

	testMux.HandleFunc("/accounts/self/groups", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		http.Error(w, "groups", http.StatusBadRequest)
	})

	_, err := CheckSelfGroupMembership(httpClient, testServer.URL, "some group")
	if err == nil {
		t.Errorf("expected an error, but was none")
	}
}
