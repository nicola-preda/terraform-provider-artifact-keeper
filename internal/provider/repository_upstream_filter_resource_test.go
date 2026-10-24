package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

func TestRepositoryUpstreamFilterEmptyPatterns(t *testing.T) {
	ctx := context.Background()
	null := types.ListNull(types.StringType)
	empty := types.ListValueMust(types.StringType, []attr.Value{})
	patterns := types.ListValueMust(types.StringType, []attr.Value{types.StringValue("^v2/test/helm/")})
	cases := []struct {
		name             string
		include, exclude types.List
	}{
		{"empty exclude", patterns, empty},
		{"empty include", empty, patterns},
		{"both empty", empty, empty},
		{"both omitted", null, null},
		{"omitted include and empty exclude", null, empty},
		{"empty include and omitted exclude", empty, null},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// An API may encode an empty pattern list as [] or null. Neither
			// representation should change the configured Terraform value.
			for _, nilResponse := range []bool{false, true} {
				name := "API empty array"
				if nilResponse {
					name = "API null"
				}
				t.Run(name, func(t *testing.T) {
					plan := repositoryUpstreamFilterResourceModel{
						ID: types.StringUnknown(), RepositoryKey: types.StringValue("test-remote"),
						IncludePatterns: tc.include, ExcludePatterns: tc.exclude, Active: types.BoolUnknown(),
					}
					apiPatterns := func(value types.List) []string {
						if len(value.Elements()) == 0 {
							if nilResponse {
								return nil
							}
							return []string{}
						}
						return []string{"^v2/test/helm/"}
					}
					response := client.UpstreamFilterResponse{
						IncludePatterns: apiPatterns(tc.include), ExcludePatterns: apiPatterns(tc.exclude),
						Active: len(tc.include.Elements())+len(tc.exclude.Elements()) > 0,
					}
					for _, operation := range []string{"create", "update", "read"} {
						t.Run(operation, func(t *testing.T) {
							var requests atomic.Int32
							r := upstreamFilterTestResource(t, response, func(req *http.Request) {
								requests.Add(1)
								wantMethod := http.MethodPut
								if operation == "read" {
									wantMethod = http.MethodGet
								}
								if req.Method != wantMethod {
									t.Errorf("method = %s, want %s", req.Method, wantMethod)
								}
								if req.Method == http.MethodPut {
									var body client.UpstreamFilter
									if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
										t.Error(err)
									}
									for name, pair := range map[string]struct {
										got  []string
										want types.List
									}{"include": {body.IncludePatterns, tc.include}, "exclude": {body.ExcludePatterns, tc.exclude}} {
										want := []string{}
										if len(pair.want.Elements()) > 0 {
											want = []string{"^v2/test/helm/"}
										}
										if !reflect.DeepEqual(pair.got, want) {
											t.Errorf("%s request patterns = %#v, want %#v", name, pair.got, want)
										}
									}
								}
							})
							var schemaResp resource.SchemaResponse
							r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
							state := tfsdk.State{Schema: schemaResp.Schema}
							assertUpstreamFilterDiagnostics(t, state.Set(ctx, plan))
							requestPlan := tfsdk.Plan{Schema: schemaResp.Schema, Raw: state.Raw}
							var result tfsdk.State
							switch operation {
							case "create":
								resp := resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
								r.Create(ctx, resource.CreateRequest{Plan: requestPlan}, &resp)
								assertUpstreamFilterDiagnostics(t, resp.Diagnostics)
								result = resp.State
							case "update":
								// Use different prior values so returning prior state instead
								// of the new plan cannot accidentally pass this regression.
								previous := plan
								previous.ID = types.StringValue("test-remote")
								previous.IncludePatterns = patterns
								previous.ExcludePatterns = patterns
								previous.Active = types.BoolValue(true)
								assertUpstreamFilterDiagnostics(t, state.Set(ctx, previous))
								resp := resource.UpdateResponse{State: state}
								r.Update(ctx, resource.UpdateRequest{Plan: requestPlan, State: state}, &resp)
								assertUpstreamFilterDiagnostics(t, resp.Diagnostics)
								result = resp.State
							case "read":
								current := plan
								current.ID = types.StringValue("test-remote")
								current.Active = types.BoolValue(response.Active)
								assertUpstreamFilterDiagnostics(t, state.Set(ctx, current))
								resp := resource.ReadResponse{State: state}
								r.Read(ctx, resource.ReadRequest{State: state}, &resp)
								assertUpstreamFilterDiagnostics(t, resp.Diagnostics)
								result = resp.State
							}
							var got repositoryUpstreamFilterResourceModel
							assertUpstreamFilterDiagnostics(t, result.Get(ctx, &got))
							if !got.IncludePatterns.Equal(tc.include) || !got.ExcludePatterns.Equal(tc.exclude) {
								t.Errorf("patterns = (%s, %s), want (%s, %s)", got.IncludePatterns, got.ExcludePatterns, tc.include, tc.exclude)
							}
							if got.ID.ValueString() != "test-remote" || got.Active.ValueBool() != response.Active {
								t.Errorf("unexpected ID or active state: %+v", got)
							}
							if requests.Load() != 1 {
								t.Errorf("API requests = %d, want 1", requests.Load())
							}
						})
					}
				})
			}
		})
	}
}

func TestRepositoryUpstreamFilterReadReportsDrift(t *testing.T) {
	ctx := context.Background()
	r := upstreamFilterTestResource(t, client.UpstreamFilterResponse{
		IncludePatterns: []string{}, ExcludePatterns: []string{"^blocked/"}, Active: true,
	}, func(*http.Request) {})
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema}
	assertUpstreamFilterDiagnostics(t, state.Set(ctx, repositoryUpstreamFilterResourceModel{
		ID: types.StringValue("test-remote"), RepositoryKey: types.StringValue("test-remote"),
		IncludePatterns: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("^old/")}),
		ExcludePatterns: types.ListNull(types.StringType), Active: types.BoolValue(true),
	}))
	resp := resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, &resp)
	assertUpstreamFilterDiagnostics(t, resp.Diagnostics)
	var got repositoryUpstreamFilterResourceModel
	assertUpstreamFilterDiagnostics(t, resp.State.Get(ctx, &got))
	wantInclude := types.ListValueMust(types.StringType, []attr.Value{})
	wantExclude := types.ListValueMust(types.StringType, []attr.Value{types.StringValue("^blocked/")})
	if !got.IncludePatterns.Equal(wantInclude) || !got.ExcludePatterns.Equal(wantExclude) {
		t.Errorf("remote drift was lost: include=%s exclude=%s", got.IncludePatterns, got.ExcludePatterns)
	}
}

func upstreamFilterTestResource(t *testing.T, response client.UpstreamFilterResponse, check func(*http.Request)) *repositoryUpstreamFilterResource {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/v1/repositories/test-remote/upstream-filter" {
			t.Errorf("unexpected request path: %s", req.URL.Path)
		}
		check(req)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(srv.Close)
	c, err := client.New(client.Config{Endpoint: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	return &repositoryUpstreamFilterResource{client: c}
}

func assertUpstreamFilterDiagnostics(t *testing.T, diagnostics diag.Diagnostics) {
	t.Helper()
	if diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
}
