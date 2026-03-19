package api

import (
	"fmt"

	goggh "github.com/cli/go-gh/v2/pkg/api"
	graphql "github.com/cli/shurcooL-graphql"
)

// Project represents a GitHub Projects v2 project.
type Project struct {
	ID          string
	Number      int
	Title       string
	Description string
	URL         string
	UpdatedAt   string
	Closed      bool
}

// Column is a named group of cards on a board view.
type Column struct {
	Name  string
	Cards []Card
}

// Card is a single item on the board.
type Card struct {
	ID        string
	Type      string // ISSUE | PULL_REQUEST | DRAFT_ISSUE
	Number    int
	Title     string
	State     string
	URL       string
	Repo      string
	Assignees []string
	Status    string
	Body      string
}

// ProjectView mirrors a saved view from GitHub Projects v2.
type ProjectView struct {
	ID           string
	Name         string
	Layout       string // BOARD_LAYOUT | TABLE_LAYOUT | ROADMAP_LAYOUT
	GroupByField string // name of the single-select field used for columns
}

// SelectField is a single-select field with ordered options.
type SelectField struct {
	ID      string
	Name    string
	Options []string // ordered option names
}

// RawItem is an ungrouped project item with all its field values.
type RawItem struct {
	ID          string
	Type        string
	IsArchived  bool
	Number      int
	Title       string
	State       string
	URL         string
	Repo        string
	Assignees   []string
	Body        string
	FieldValues map[string]string // field name → selected option name
}

// BoardData holds everything fetched from a project in one call.
// Grouping into columns is done client-side via GroupByField.
type BoardData struct {
	Views  []ProjectView
	Fields []SelectField
	Items  []RawItem
}

// GroupByField buckets RawItems into Columns using the named single-select field.
// Column order follows the field's option order. Falls back to "No <field>" bucket.
func GroupByField(data *BoardData, fieldName string) []Column {
	// Find the field's ordered options.
	var options []string
	for _, f := range data.Fields {
		if f.Name == fieldName {
			options = f.Options
			break
		}
	}
	noGroup := "No " + fieldName
	if len(options) == 0 {
		options = []string{noGroup}
	}

	buckets := make(map[string][]Card, len(options)+1)
	for _, opt := range options {
		buckets[opt] = nil
	}

	for _, item := range data.Items {
		if item.IsArchived {
			continue
		}
		card := Card{
			ID:        item.ID,
			Type:      item.Type,
			Number:    item.Number,
			Title:     item.Title,
			State:     item.State,
			URL:       item.URL,
			Repo:      item.Repo,
			Assignees: item.Assignees,
			Body:      item.Body,
		}

		val := item.FieldValues[fieldName]
		if val == "" {
			val = noGroup
		}
		card.Status = val

		if _, ok := buckets[val]; ok {
			buckets[val] = append(buckets[val], card)
		} else {
			buckets[noGroup] = append(buckets[noGroup], card)
		}
	}

	columns := make([]Column, 0, len(options))
	for _, opt := range options {
		columns = append(columns, Column{Name: opt, Cards: buckets[opt]})
	}
	// Append the no-group bucket only if it has cards.
	if cards := buckets[noGroup]; len(cards) > 0 {
		// Avoid duplicate if noGroup was already an option.
		found := false
		for _, opt := range options {
			if opt == noGroup {
				found = true
				break
			}
		}
		if !found {
			columns = append(columns, Column{Name: noGroup, Cards: cards})
		}
	}
	return columns
}

// Client wraps the go-gh GraphQL client.
type Client struct {
	gql *goggh.GraphQLClient
}

// NewClient creates a Client using the current gh auth context.
func NewClient() (*Client, error) {
	gql, err := goggh.DefaultGraphQLClient()
	if err != nil {
		return nil, fmt.Errorf("creating GraphQL client: %w", err)
	}
	return &Client{gql: gql}, nil
}

// ViewerLogin returns the authenticated user's login.
func (c *Client) ViewerLogin() (string, error) {
	var q struct {
		Viewer struct {
			Login string
		}
	}
	if err := c.gql.Query("ViewerLogin", &q, nil); err != nil {
		return "", err
	}
	return q.Viewer.Login, nil
}

// ListUserProjects fetches up to 50 open projects for the given user login.
func (c *Client) ListUserProjects(login string) ([]Project, error) {
	var q struct {
		User struct {
			ProjectsV2 struct {
				Nodes []struct {
					ID               string
					Number           int
					Title            string
					ShortDescription string
					Closed           bool
					URL              string
					UpdatedAt        string
				}
			} `graphql:"projectsV2(first: 50)"`
		} `graphql:"user(login: $login)"`
	}
	variables := map[string]interface{}{
		"login": graphql.String(login),
	}
	if err := c.gql.Query("ListUserProjects", &q, variables); err != nil {
		return nil, err
	}
	var projects []Project
	for _, n := range q.User.ProjectsV2.Nodes {
		if n.Closed {
			continue
		}
		projects = append(projects, Project{
			ID:          n.ID,
			Number:      n.Number,
			Title:       n.Title,
			Description: n.ShortDescription,
			URL:         n.URL,
			UpdatedAt:   n.UpdatedAt,
			Closed:      n.Closed,
		})
	}
	return projects, nil
}

// ListOrgProjects fetches up to 50 open projects for the given org login.
func (c *Client) ListOrgProjects(org string) ([]Project, error) {
	var q struct {
		Organization struct {
			ProjectsV2 struct {
				Nodes []struct {
					ID               string
					Number           int
					Title            string
					ShortDescription string
					Closed           bool
					URL              string
					UpdatedAt        string
				}
			} `graphql:"projectsV2(first: 50)"`
		} `graphql:"organization(login: $org)"`
	}
	variables := map[string]interface{}{
		"org": graphql.String(org),
	}
	if err := c.gql.Query("ListOrgProjects", &q, variables); err != nil {
		return nil, err
	}
	var projects []Project
	for _, n := range q.Organization.ProjectsV2.Nodes {
		if n.Closed {
			continue
		}
		projects = append(projects, Project{
			ID:          n.ID,
			Number:      n.Number,
			Title:       n.Title,
			Description: n.ShortDescription,
			URL:         n.URL,
			UpdatedAt:   n.UpdatedAt,
			Closed:      n.Closed,
		})
	}
	return projects, nil
}

// FlatItems returns all non-archived items in API order, for table views.
func FlatItems(data *BoardData) []Card {
	var cards []Card
	for _, item := range data.Items {
		if item.IsArchived {
			continue
		}
		cards = append(cards, Card{
			ID:        item.ID,
			Type:      item.Type,
			Number:    item.Number,
			Title:     item.Title,
			State:     item.State,
			URL:       item.URL,
			Repo:      item.Repo,
			Assignees: item.Assignees,
			Body:      item.Body,
			Status:    item.FieldValues["Status"],
		})
	}
	return cards
}

// ListViewerOrgs returns the login names of orgs the authenticated user belongs to.
func (c *Client) ListViewerOrgs() ([]string, error) {
	var q struct {
		Viewer struct {
			Organizations struct {
				Nodes []struct{ Login string }
			} `graphql:"organizations(first: 20)"`
		}
	}
	if err := c.gql.Query("ListViewerOrgs", &q, nil); err != nil {
		return nil, err
	}
	var orgs []string
	for _, n := range q.Viewer.Organizations.Nodes {
		orgs = append(orgs, n.Login)
	}
	return orgs, nil
}

// GetProjectBoard fetches views, fields, and all items for a project.
// Use GroupByField to render a specific view's columns.
func (c *Client) GetProjectBoard(projectID string) (*BoardData, error) {
	var q struct {
		Node struct {
			Project struct {
				Views struct {
					Nodes []struct {
						ID     string
						Name   string
						Layout string
						GroupByFields struct {
							Nodes []struct {
								AsSelectField struct {
									Name string
								} `graphql:"... on ProjectV2SingleSelectField"`
							}
						} `graphql:"groupByFields(first: 5)"`
					}
				} `graphql:"views(first: 20)"`
				Fields struct {
					Nodes []struct {
						AsSelectField struct {
							ID      string
							Name    string
							Options []struct {
								ID   string
								Name string
							}
						} `graphql:"... on ProjectV2SingleSelectField"`
					}
				} `graphql:"fields(first: 20)"`
				Items struct {
					Nodes []struct {
						ID         string
						Type       string
						IsArchived bool
						Content    struct {
							AsIssue struct {
								Number     int
								Title      string
								State      string
								URL        string
								Repository struct {
									NameWithOwner string
								}
								Assignees struct {
									Nodes []struct{ Login string }
								} `graphql:"assignees(first: 5)"`
							} `graphql:"... on Issue"`
							AsPullRequest struct {
								Number     int
								Title      string
								State      string
								URL        string
								Repository struct {
									NameWithOwner string
								}
								Assignees struct {
									Nodes []struct{ Login string }
								} `graphql:"assignees(first: 5)"`
							} `graphql:"... on PullRequest"`
							AsDraftIssue struct {
								Title string
								Body  string
							} `graphql:"... on DraftIssue"`
						}
						FieldValues struct {
							Nodes []struct {
								SingleSelectValue struct {
									Name  string
									Field struct {
										AsSelectField struct {
											Name string
										} `graphql:"... on ProjectV2SingleSelectField"`
									}
								} `graphql:"... on ProjectV2ItemFieldSingleSelectValue"`
							}
						} `graphql:"fieldValues(first: 20)"`
					}
				} `graphql:"items(first: 100)"`
			} `graphql:"... on ProjectV2"`
		} `graphql:"node(id: $id)"`
	}

	variables := map[string]interface{}{
		"id": graphql.ID(projectID),
	}
	if err := c.gql.Query("GetProjectBoard", &q, variables); err != nil {
		return nil, err
	}

	// Build SelectFields map.
	var fields []SelectField
	for _, f := range q.Node.Project.Fields.Nodes {
		sf := f.AsSelectField
		if sf.Name == "" {
			continue
		}
		field := SelectField{ID: sf.ID, Name: sf.Name}
		for _, opt := range sf.Options {
			field.Options = append(field.Options, opt.Name)
		}
		fields = append(fields, field)
	}

	// Build views, resolving groupByField name.
	var views []ProjectView
	for _, v := range q.Node.Project.Views.Nodes {
		pv := ProjectView{
			ID:     v.ID,
			Name:   v.Name,
			Layout: v.Layout,
		}
		// Use the first groupByField as the column grouping field.
		for _, gf := range v.GroupByFields.Nodes {
			if gf.AsSelectField.Name != "" {
				pv.GroupByField = gf.AsSelectField.Name
				break
			}
		}
		// Only include board views that have a groupByField,
		// or fall back gracefully for other layouts.
		views = append(views, pv)
	}

	// If no views returned, synthesise a default "Board" view grouped by Status.
	if len(views) == 0 {
		views = []ProjectView{{Name: "Board", Layout: "BOARD_LAYOUT", GroupByField: "Status"}}
	}

	// Build raw items.
	var items []RawItem
	for _, item := range q.Node.Project.Items.Nodes {
		ri := RawItem{
			ID:          item.ID,
			Type:        item.Type,
			IsArchived:  item.IsArchived,
			FieldValues: make(map[string]string),
		}
		switch item.Type {
		case "ISSUE":
			ri.Number = item.Content.AsIssue.Number
			ri.Title = item.Content.AsIssue.Title
			ri.State = item.Content.AsIssue.State
			ri.URL = item.Content.AsIssue.URL
			ri.Repo = item.Content.AsIssue.Repository.NameWithOwner
			for _, a := range item.Content.AsIssue.Assignees.Nodes {
				ri.Assignees = append(ri.Assignees, "@"+a.Login)
			}
		case "PULL_REQUEST":
			ri.Number = item.Content.AsPullRequest.Number
			ri.Title = item.Content.AsPullRequest.Title
			ri.State = item.Content.AsPullRequest.State
			ri.URL = item.Content.AsPullRequest.URL
			ri.Repo = item.Content.AsPullRequest.Repository.NameWithOwner
			for _, a := range item.Content.AsPullRequest.Assignees.Nodes {
				ri.Assignees = append(ri.Assignees, "@"+a.Login)
			}
		case "DRAFT_ISSUE":
			ri.Title = item.Content.AsDraftIssue.Title
			ri.Body = item.Content.AsDraftIssue.Body
		}
		for _, fv := range item.FieldValues.Nodes {
			fieldName := fv.SingleSelectValue.Field.AsSelectField.Name
			val := fv.SingleSelectValue.Name
			if fieldName != "" && val != "" {
				ri.FieldValues[fieldName] = val
			}
		}
		items = append(items, ri)
	}

	return &BoardData{Views: views, Fields: fields, Items: items}, nil
}
