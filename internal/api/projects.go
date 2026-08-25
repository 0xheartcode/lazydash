// Package api is the GitHub Projects v2 data source: it wraps the go-gh GraphQL
// client and maps GitHub's schema onto lazydash's backend-neutral core model.
package api

import (
	"fmt"
	"strconv"

	"github.com/0xheartcode/lazydash/internal/core"
	goggh "github.com/cli/go-gh/v2/pkg/api"
	graphql "github.com/cli/shurcooL-graphql"
)

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
func (c *Client) ListUserProjects(login string) ([]core.Project, error) {
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
	var projects []core.Project
	for _, n := range q.User.ProjectsV2.Nodes {
		if n.Closed {
			continue
		}
		projects = append(projects, core.Project{
			ID:          n.ID,
			Number:      n.Number,
			Title:       n.Title,
			Owner:       login,
			Description: n.ShortDescription,
			URL:         n.URL,
			UpdatedAt:   n.UpdatedAt,
		})
	}
	return projects, nil
}

// ListOrgProjects fetches up to 50 open projects for the given org login.
func (c *Client) ListOrgProjects(org string) ([]core.Project, error) {
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
	var projects []core.Project
	for _, n := range q.Organization.ProjectsV2.Nodes {
		if n.Closed {
			continue
		}
		projects = append(projects, core.Project{
			ID:          n.ID,
			Number:      n.Number,
			Title:       n.Title,
			Owner:       org,
			Description: n.ShortDescription,
			URL:         n.URL,
			UpdatedAt:   n.UpdatedAt,
		})
	}
	return projects, nil
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
// Use core.GroupByField to render a specific view's columns.
func (c *Client) GetProjectBoard(projectID string) (*core.BoardData, error) {
	var q struct {
		Node struct {
			Project struct {
				Views struct {
					Nodes []struct {
						ID            string
						Name          string
						Layout        string
						GroupByFields struct {
							Nodes []struct {
								AsSelectField struct {
									Name string
								} `graphql:"... on ProjectV2SingleSelectField"`
							}
						} `graphql:"groupByFields(first: 5)"`
						VisibleFields struct {
							Nodes []struct {
								ID       string
								Name     string
								DataType string
							}
						} `graphql:"visibleFields(first: 20)"`
					}
				} `graphql:"views(first: 20)"`
				Fields struct {
					Nodes []struct {
						AsSelectField struct {
							ID      string
							Name    string
							Options []struct {
								ID    string
								Name  string
								Color string
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
								TextValue struct {
									Text  string
									Field struct {
										AsField struct {
											Name string
										} `graphql:"... on ProjectV2Field"`
									}
								} `graphql:"... on ProjectV2ItemFieldTextValue"`
								NumberValue struct {
									Number float64
									Field  struct {
										AsField struct {
											Name string
										} `graphql:"... on ProjectV2Field"`
									}
								} `graphql:"... on ProjectV2ItemFieldNumberValue"`
								DateValue struct {
									Date  string
									Field struct {
										AsField struct {
											Name string
										} `graphql:"... on ProjectV2Field"`
									}
								} `graphql:"... on ProjectV2ItemFieldDateValue"`
								IterationValue struct {
									Title string
									Field struct {
										AsIterField struct {
											Name string
										} `graphql:"... on ProjectV2IterationField"`
									}
								} `graphql:"... on ProjectV2ItemFieldIterationValue"`
								MilestoneValue struct {
									Milestone struct {
										Title string
									}
									Field struct {
										AsField struct {
											Name string
										} `graphql:"... on ProjectV2Field"`
									}
								} `graphql:"... on ProjectV2ItemFieldMilestoneValue"`
								LabelValue struct {
									Labels struct {
										Nodes []struct {
											Name  string
											Color string
										}
									} `graphql:"labels(first: 10)"`
								} `graphql:"... on ProjectV2ItemFieldLabelValue"`
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
	var fields []core.SelectField
	for _, f := range q.Node.Project.Fields.Nodes {
		sf := f.AsSelectField
		if sf.Name == "" {
			continue
		}
		field := core.SelectField{ID: sf.ID, Name: sf.Name}
		for _, opt := range sf.Options {
			field.Options = append(field.Options, core.FieldOption{Name: opt.Name, Color: opt.Color})
		}
		fields = append(fields, field)
	}

	// Build views, resolving groupByField name and visible fields.
	var views []core.ProjectView
	for _, v := range q.Node.Project.Views.Nodes {
		pv := core.ProjectView{
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
		// Parse visible fields for this view.
		for _, vf := range v.VisibleFields.Nodes {
			if vf.Name != "" {
				pv.VisibleFields = append(pv.VisibleFields, core.VisibleField{ID: vf.ID, Name: vf.Name, DataType: vf.DataType})
			}
		}
		views = append(views, pv)
	}

	// If no views returned, synthesise a default "Board" view grouped by Status.
	if len(views) == 0 {
		views = []core.ProjectView{{Name: "Board", Layout: "BOARD_LAYOUT", GroupByField: "Status"}}
	}

	// Build raw items.
	var items []core.RawItem
	for _, item := range q.Node.Project.Items.Nodes {
		ri := core.RawItem{
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
			if fn := fv.SingleSelectValue.Field.AsSelectField.Name; fn != "" {
				if v := fv.SingleSelectValue.Name; v != "" {
					ri.FieldValues[fn] = v
				}
			} else if fn := fv.TextValue.Field.AsField.Name; fn != "" {
				if v := fv.TextValue.Text; v != "" {
					ri.FieldValues[fn] = v
				}
			} else if fn := fv.NumberValue.Field.AsField.Name; fn != "" {
				ri.FieldValues[fn] = strconv.FormatFloat(fv.NumberValue.Number, 'f', -1, 64)
			} else if fn := fv.DateValue.Field.AsField.Name; fn != "" {
				if v := fv.DateValue.Date; v != "" {
					ri.FieldValues[fn] = v
				}
			} else if fn := fv.IterationValue.Field.AsIterField.Name; fn != "" {
				if v := fv.IterationValue.Title; v != "" {
					ri.FieldValues[fn] = v
				}
			} else if fn := fv.MilestoneValue.Field.AsField.Name; fn != "" {
				if v := fv.MilestoneValue.Milestone.Title; v != "" {
					ri.FieldValues[fn] = v
				}
			} else if len(fv.LabelValue.Labels.Nodes) > 0 {
				for _, l := range fv.LabelValue.Labels.Nodes {
					ri.Labels = append(ri.Labels, core.Label{Name: l.Name, Color: l.Color})
				}
			}
		}
		items = append(items, ri)
	}

	return &core.BoardData{Views: views, Fields: fields, Items: items}, nil
}
