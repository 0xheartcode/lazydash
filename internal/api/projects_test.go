package api

import (
	"io"
	"net/http"
	"strings"
	"testing"

	goggh "github.com/cli/go-gh/v2/pkg/api"
)

// stubTransport returns a fixed GraphQL response for every request, which lets
// the tests drive GetProjectBoard without any network access.
type stubTransport struct {
	status int
	body   string
}

func (s stubTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	status := s.status
	if status == 0 {
		status = http.StatusOK
	}
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Body:       io.NopCloser(strings.NewReader(s.body)),
		Header:     make(http.Header),
	}, nil
}

// newStubClient builds a Client whose GraphQL transport always answers with the
// supplied canned payload. Host and token are set so the go-gh client skips its
// environment resolution and never touches the real gh configuration.
func newStubClient(t *testing.T, status int, body string) *Client {
	t.Helper()
	gql, err := goggh.NewGraphQLClient(goggh.ClientOptions{
		Host:      "github.com",
		AuthToken: "test-token",
		Transport: stubTransport{status: status, body: body},
	})
	if err != nil {
		t.Fatalf("building stub GraphQL client: %v", err)
	}
	return &Client{gql: gql}
}

// fullBoardResponse is a canned GraphQL payload describing a project with one
// view, one single-select field, and an issue, a pull request, and a draft.
const fullBoardResponse = `{"data":{"node":{
  "views":{"nodes":[
    {"id":"V1","name":"Board","layout":"BOARD_LAYOUT",
     "groupByFields":{"nodes":[{"name":"Status"}]},
     "visibleFields":{"nodes":[
       {"id":"VF1","name":"Title","dataType":"TITLE"},
       {"id":"VF2","name":"Status","dataType":"SINGLE_SELECT"}
     ]}}
  ]},
  "fields":{"nodes":[
    {"id":"F1","name":"Status","options":[
      {"id":"O1","name":"Todo","color":"GRAY"},
      {"id":"O2","name":"Done","color":"GREEN"}
    ]},
    {}
  ]},
  "items":{"nodes":[
    {"id":"I1","type":"ISSUE","isArchived":false,
     "content":{"number":7,"title":"Fix bug","state":"OPEN","url":"https://example.test/i/7",
       "repository":{"nameWithOwner":"acme/widgets"},
       "assignees":{"nodes":[{"login":"alice"}]}},
     "fieldValues":{"nodes":[{"name":"Todo","field":{"name":"Status"}}]}},
    {"id":"P1","type":"PULL_REQUEST","isArchived":false,
     "content":{"number":9,"title":"Add feature","state":"MERGED","url":"https://example.test/p/9",
       "repository":{"nameWithOwner":"acme/widgets"},
       "assignees":{"nodes":[{"login":"bob"},{"login":"carol"}]}},
     "fieldValues":{"nodes":[{"name":"Done","field":{"name":"Status"}}]}},
    {"id":"D1","type":"DRAFT_ISSUE","isArchived":true,
     "content":{"title":"Rough idea","body":"needs refining"},
     "fieldValues":{"nodes":[]}}
  ]}
}}}`

func TestGetProjectBoardParsesResponse(t *testing.T) {
	c := newStubClient(t, http.StatusOK, fullBoardResponse)

	bd, err := c.GetProjectBoard("PROJECT_ID")
	if err != nil {
		t.Fatalf("GetProjectBoard returned error: %v", err)
	}

	// Views.
	if len(bd.Views) != 1 {
		t.Fatalf("got %d views, want 1", len(bd.Views))
	}
	v := bd.Views[0]
	if v.ID != "V1" || v.Name != "Board" || v.Layout != "BOARD_LAYOUT" {
		t.Errorf("view header = %+v, want ID V1 / Board / BOARD_LAYOUT", v)
	}
	if v.GroupByField != "Status" {
		t.Errorf("GroupByField = %q, want Status", v.GroupByField)
	}
	if len(v.VisibleFields) != 2 {
		t.Fatalf("got %d visible fields, want 2", len(v.VisibleFields))
	}
	if v.VisibleFields[0].Name != "Title" || v.VisibleFields[0].DataType != "TITLE" {
		t.Errorf("visible field 0 = %+v, want Title/TITLE", v.VisibleFields[0])
	}
	if v.VisibleFields[1].ID != "VF2" || v.VisibleFields[1].DataType != "SINGLE_SELECT" {
		t.Errorf("visible field 1 = %+v, want VF2/SINGLE_SELECT", v.VisibleFields[1])
	}

	// Fields: the empty node must be skipped, leaving one select field.
	if len(bd.Fields) != 1 {
		t.Fatalf("got %d fields, want 1", len(bd.Fields))
	}
	f := bd.Fields[0]
	if f.ID != "F1" || f.Name != "Status" {
		t.Errorf("field header = %+v, want F1/Status", f)
	}
	if len(f.Options) != 2 {
		t.Fatalf("got %d options, want 2", len(f.Options))
	}
	if f.Options[0].ID != "O1" || f.Options[0].Name != "Todo" || f.Options[0].Color != "GRAY" {
		t.Errorf("option 0 = %+v, want O1/Todo/GRAY", f.Options[0])
	}
	if f.Options[1].Name != "Done" || f.Options[1].Color != "GREEN" {
		t.Errorf("option 1 = %+v, want Done/GREEN", f.Options[1])
	}

	// Items.
	if len(bd.Items) != 3 {
		t.Fatalf("got %d items, want 3", len(bd.Items))
	}

	issue := bd.Items[0]
	if issue.Type != "ISSUE" || issue.Number != 7 || issue.Title != "Fix bug" {
		t.Errorf("issue core = %+v, want ISSUE/7/Fix bug", issue)
	}
	if issue.State != "OPEN" || issue.URL != "https://example.test/i/7" || issue.Repo != "acme/widgets" {
		t.Errorf("issue meta = %+v", issue)
	}
	if len(issue.Assignees) != 1 || issue.Assignees[0] != "@alice" {
		t.Errorf("issue assignees = %v, want [@alice]", issue.Assignees)
	}
	if issue.FieldValues["Status"] != "Todo" {
		t.Errorf("issue Status = %q, want Todo", issue.FieldValues["Status"])
	}

	pr := bd.Items[1]
	if pr.Type != "PULL_REQUEST" || pr.Number != 9 || pr.State != "MERGED" {
		t.Errorf("pr core = %+v, want PULL_REQUEST/9/MERGED", pr)
	}
	if len(pr.Assignees) != 2 || pr.Assignees[0] != "@bob" || pr.Assignees[1] != "@carol" {
		t.Errorf("pr assignees = %v, want [@bob @carol]", pr.Assignees)
	}
	if pr.FieldValues["Status"] != "Done" {
		t.Errorf("pr Status = %q, want Done", pr.FieldValues["Status"])
	}

	draft := bd.Items[2]
	if draft.Type != "DRAFT_ISSUE" || draft.Title != "Rough idea" || draft.Body != "needs refining" {
		t.Errorf("draft = %+v, want DRAFT_ISSUE/Rough idea/needs refining", draft)
	}
	if !draft.IsArchived {
		t.Error("draft IsArchived = false, want true")
	}
	if len(draft.FieldValues) != 0 {
		t.Errorf("draft FieldValues = %v, want empty", draft.FieldValues)
	}
}

func TestGetProjectBoardSynthesisesDefaultView(t *testing.T) {
	const noViews = `{"data":{"node":{
      "views":{"nodes":[]},
      "fields":{"nodes":[]},
      "items":{"nodes":[
        {"id":"I1","type":"ISSUE","isArchived":false,
         "content":{"number":1,"title":"Only item","state":"OPEN","url":"https://example.test/i/1",
           "repository":{"nameWithOwner":"acme/widgets"},"assignees":{"nodes":[]}},
         "fieldValues":{"nodes":[]}}
      ]}
    }}}`
	c := newStubClient(t, http.StatusOK, noViews)

	bd, err := c.GetProjectBoard("PROJECT_ID")
	if err != nil {
		t.Fatalf("GetProjectBoard returned error: %v", err)
	}
	if len(bd.Views) != 1 {
		t.Fatalf("got %d views, want 1 synthesised", len(bd.Views))
	}
	v := bd.Views[0]
	if v.Name != "Board" || v.Layout != "BOARD_LAYOUT" || v.GroupByField != "Status" {
		t.Errorf("synthesised view = %+v, want Board/BOARD_LAYOUT/Status", v)
	}
	if len(bd.Items) != 1 || bd.Items[0].Title != "Only item" {
		t.Errorf("items = %+v, want a single Only item", bd.Items)
	}
}

func TestGetProjectBoardPropagatesHTTPError(t *testing.T) {
	c := newStubClient(t, http.StatusInternalServerError, `{"message":"boom"}`)

	if _, err := c.GetProjectBoard("PROJECT_ID"); err == nil {
		t.Error("GetProjectBoard should return an error on a non-200 response")
	}
}

func TestGetProjectBoardPropagatesGraphQLError(t *testing.T) {
	c := newStubClient(t, http.StatusOK, `{"errors":[{"message":"bad query"}]}`)

	if _, err := c.GetProjectBoard("PROJECT_ID"); err == nil {
		t.Error("GetProjectBoard should return an error when the payload carries GraphQL errors")
	}
}

func TestViewerLogin(t *testing.T) {
	c := newStubClient(t, http.StatusOK, `{"data":{"viewer":{"login":"octocat"}}}`)

	login, err := c.ViewerLogin()
	if err != nil {
		t.Fatalf("ViewerLogin returned error: %v", err)
	}
	if login != "octocat" {
		t.Errorf("login = %q, want octocat", login)
	}
}

func TestListUserProjectsSkipsClosed(t *testing.T) {
	const body = `{"data":{"user":{"projectsV2":{"nodes":[
      {"id":"P1","number":1,"title":"Open one","shortDescription":"desc","closed":false,"url":"https://example.test/p/1","updatedAt":"2026-01-02"},
      {"id":"P2","number":2,"title":"Closed one","closed":true}
    ]}}}}`
	c := newStubClient(t, http.StatusOK, body)

	projects, err := c.ListUserProjects("octocat")
	if err != nil {
		t.Fatalf("ListUserProjects returned error: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("got %d projects, want 1 (closed skipped)", len(projects))
	}
	p := projects[0]
	if p.ID != "P1" || p.Number != 1 || p.Title != "Open one" {
		t.Errorf("project core = %+v, want P1/1/Open one", p)
	}
	if p.Owner != "octocat" || p.Description != "desc" || p.URL != "https://example.test/p/1" || p.UpdatedAt != "2026-01-02" {
		t.Errorf("project meta = %+v", p)
	}
}

func TestListOrgProjectsSkipsClosed(t *testing.T) {
	const body = `{"data":{"organization":{"projectsV2":{"nodes":[
      {"id":"P1","number":1,"title":"Open","closed":false,"url":"https://example.test/p/1","updatedAt":"2026-01-02"},
      {"id":"P2","number":2,"title":"Closed","closed":true}
    ]}}}}`
	c := newStubClient(t, http.StatusOK, body)

	projects, err := c.ListOrgProjects("acme")
	if err != nil {
		t.Fatalf("ListOrgProjects returned error: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("got %d projects, want 1 (closed skipped)", len(projects))
	}
	if projects[0].Owner != "acme" {
		t.Errorf("Owner = %q, want acme", projects[0].Owner)
	}
}

func TestListViewerOrgs(t *testing.T) {
	const body = `{"data":{"viewer":{"organizations":{"nodes":[{"login":"acme"},{"login":"globex"}]}}}}`
	c := newStubClient(t, http.StatusOK, body)

	orgs, err := c.ListViewerOrgs()
	if err != nil {
		t.Fatalf("ListViewerOrgs returned error: %v", err)
	}
	if len(orgs) != 2 || orgs[0] != "acme" || orgs[1] != "globex" {
		t.Errorf("orgs = %v, want [acme globex]", orgs)
	}
}
