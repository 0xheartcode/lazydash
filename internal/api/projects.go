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

// Column is a status column on a project board.
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

// GetProjectBoard fetches all items for a project and groups them by Status.
func (c *Client) GetProjectBoard(projectID string) ([]Column, error) {
	var q struct {
		Node struct {
			Project struct {
				Fields struct {
					Nodes []struct {
						SingleSelectField struct {
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
										Name string
									} `graphql:"... on ProjectV2SingleSelectField"`
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

	// Find the Status single-select field and its ordered options.
	var statusOptions []string
	for _, f := range q.Node.Project.Fields.Nodes {
		if f.SingleSelectField.Name == "Status" {
			for _, opt := range f.SingleSelectField.Options {
				statusOptions = append(statusOptions, opt.Name)
			}
			break
		}
	}
	if len(statusOptions) == 0 {
		statusOptions = []string{"No Status"}
	}

	// Bucket cards by status.
	buckets := make(map[string][]Card, len(statusOptions))
	for _, opt := range statusOptions {
		buckets[opt] = nil
	}

	for _, item := range q.Node.Project.Items.Nodes {
		if item.IsArchived {
			continue
		}
		card := Card{ID: item.ID, Type: item.Type}
		switch item.Type {
		case "ISSUE":
			card.Number = item.Content.AsIssue.Number
			card.Title = item.Content.AsIssue.Title
			card.State = item.Content.AsIssue.State
			card.URL = item.Content.AsIssue.URL
			card.Repo = item.Content.AsIssue.Repository.NameWithOwner
			for _, a := range item.Content.AsIssue.Assignees.Nodes {
				card.Assignees = append(card.Assignees, "@"+a.Login)
			}
		case "PULL_REQUEST":
			card.Number = item.Content.AsPullRequest.Number
			card.Title = item.Content.AsPullRequest.Title
			card.State = item.Content.AsPullRequest.State
			card.URL = item.Content.AsPullRequest.URL
			card.Repo = item.Content.AsPullRequest.Repository.NameWithOwner
			for _, a := range item.Content.AsPullRequest.Assignees.Nodes {
				card.Assignees = append(card.Assignees, "@"+a.Login)
			}
		case "DRAFT_ISSUE":
			card.Title = item.Content.AsDraftIssue.Title
			card.Body = item.Content.AsDraftIssue.Body
		}

		// Find status from fieldValues.
		status := "No Status"
		for _, fv := range item.FieldValues.Nodes {
			if fv.SingleSelectValue.Field.Name == "Status" && fv.SingleSelectValue.Name != "" {
				status = fv.SingleSelectValue.Name
				break
			}
		}
		card.Status = status

		if _, ok := buckets[status]; ok {
			buckets[status] = append(buckets[status], card)
		} else {
			buckets["No Status"] = append(buckets["No Status"], card)
		}
	}

	// Build columns in the order defined by status options.
	columns := make([]Column, 0, len(statusOptions))
	for _, opt := range statusOptions {
		columns = append(columns, Column{Name: opt, Cards: buckets[opt]})
	}
	return columns, nil
}

