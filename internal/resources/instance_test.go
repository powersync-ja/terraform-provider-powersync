package resources

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/powersync/terraform-provider-powersync/internal/client"
)

// ── Value helpers ─────────────────────────────────────────────────────────────

func TestStrVal(t *testing.T) {
	tests := []struct {
		name string
		in   types.String
		want string
	}{
		{"null → empty", types.StringNull(), ""},
		{"unknown → empty", types.StringUnknown(), ""},
		{"empty value → empty", types.StringValue(""), ""},
		{"plain → plain", types.StringValue("hello"), "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := strVal(tt.in); got != tt.want {
				t.Errorf("strVal(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestStringOrNull(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want types.String
	}{
		{"empty → null", "", types.StringNull()},
		{"non-empty → value", "x", types.StringValue("x")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stringOrNull(tt.in); !got.Equal(tt.want) {
				t.Errorf("stringOrNull(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestBoolOrNull(t *testing.T) {
	tests := []struct {
		name string
		in   bool
		want types.Bool
	}{
		{"false → null (treated as unset)", false, types.BoolNull()},
		{"true → BoolValue(true)", true, types.BoolValue(true)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := boolOrNull(tt.in); !got.Equal(tt.want) {
				t.Errorf("boolOrNull(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestInt64OrNull(t *testing.T) {
	five := int64(5)
	zero := int64(0)
	tests := []struct {
		name string
		in   *int64
		want types.Int64
	}{
		{"nil → null", nil, types.Int64Null()},
		{"zero → Int64Value(0)", &zero, types.Int64Value(0)},
		{"five → Int64Value(5)", &five, types.Int64Value(5)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := int64OrNull(tt.in); !got.Equal(tt.want) {
				t.Errorf("int64OrNull(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

// ── connectionModelToClient ───────────────────────────────────────────────────

func TestConnectionModelToClient_URIForm(t *testing.T) {
	got := connectionModelToClient(connectionModel{
		Type: types.StringValue("postgresql"),
		URI:  types.StringValue("postgresql://u:p@h:5432/db"),
		// All other fields null — verify they don't leak into the output.
	})

	if got.Type != "postgresql" {
		t.Errorf("Type = %q, want %q", got.Type, "postgresql")
	}
	if got.URI != "postgresql://u:p@h:5432/db" {
		t.Errorf("URI = %q", got.URI)
	}
	if got.Hostname != "" || got.Database != "" || got.Username != "" {
		t.Errorf("non-URI fields leaked: hostname=%q, database=%q, username=%q", got.Hostname, got.Database, got.Username)
	}
	if got.Port != nil {
		t.Errorf("Port should be nil when not set, got %v", *got.Port)
	}
	if got.Password != nil {
		t.Errorf("Password should be nil when not set, got %v", got.Password)
	}
}

func TestConnectionModelToClient_IndividualFields(t *testing.T) {
	got := connectionModelToClient(connectionModel{
		Type:     types.StringValue("postgresql"),
		Name:     types.StringValue("supabase-main"),
		Hostname: types.StringValue("db.example.com"),
		Port:     types.Int64Value(5432),
		Username: types.StringValue("powersync_role"),
		Password: types.StringValue("secret"),
		Database: types.StringValue("postgres"),
		SSLMode:  types.StringValue("verify-full"),
	})

	if got.Name != "supabase-main" {
		t.Errorf("Name = %q", got.Name)
	}
	if got.Port == nil || *got.Port != 5432 {
		t.Errorf("Port = %v, want 5432", got.Port)
	}
	if got.Password == nil {
		t.Fatal("Password should be wrapped in HostedSecret, got nil")
	}
	if got.Password.Secret != "secret" {
		t.Errorf("Password.Secret = %q, want %q", got.Password.Secret, "secret")
	}
	if got.SSLMode != "verify-full" {
		t.Errorf("SSLMode = %q", got.SSLMode)
	}
}

func TestConnectionModelToClient_EmptyPasswordNotWrapped(t *testing.T) {
	// An empty-string password should NOT be sent as a HostedSecret — that would
	// overwrite the server-side stored secret with empty string.
	got := connectionModelToClient(connectionModel{
		Type:     types.StringValue("postgresql"),
		Password: types.StringValue(""),
	})
	if got.Password != nil {
		t.Errorf("empty Password should be nil (omitted), got %+v", got.Password)
	}
}

func TestConnectionModelToClient_ClientCertAndKey(t *testing.T) {
	got := connectionModelToClient(connectionModel{
		Type:              types.StringValue("postgresql"),
		ClientCertificate: types.StringValue("---CERT---"),
		ClientPrivateKey:  types.StringValue("---KEY---"),
	})
	if got.ClientCertificate != "---CERT---" {
		t.Errorf("ClientCertificate = %q", got.ClientCertificate)
	}
	if got.ClientPrivateKey == nil || got.ClientPrivateKey.Secret != "---KEY---" {
		t.Errorf("ClientPrivateKey not wrapped correctly, got %+v", got.ClientPrivateKey)
	}
}

// ── clientAuthModelToClient ───────────────────────────────────────────────────

func TestClientAuthModelToClient_SupabaseOnly(t *testing.T) {
	got := clientAuthModelToClient(clientAuthModel{
		Supabase:             types.BoolValue(true),
		AllowTemporaryTokens: types.BoolValue(true),
		AdditionalAudiences:  types.ListNull(types.StringType),
	})
	if !got.Supabase {
		t.Errorf("Supabase = false, want true")
	}
	if !got.AllowTemporaryTokens {
		t.Errorf("AllowTemporaryTokens = false, want true")
	}
	if got.JWKSUri != "" {
		t.Errorf("JWKSUri = %q, want empty", got.JWKSUri)
	}
	if got.AdditionalAudiences != nil {
		t.Errorf("AdditionalAudiences = %v, want nil when list is null", got.AdditionalAudiences)
	}
}

func TestClientAuthModelToClient_JWKSWithAudiences(t *testing.T) {
	audiences, _ := types.ListValueFrom(context.Background(), types.StringType, []string{"aud1", "aud2"})
	got := clientAuthModelToClient(clientAuthModel{
		JWKSUri:             types.StringValue("https://auth.example.com/jwks.json"),
		AdditionalAudiences: audiences,
	})
	if got.JWKSUri != "https://auth.example.com/jwks.json" {
		t.Errorf("JWKSUri = %q", got.JWKSUri)
	}
	if !reflect.DeepEqual(got.AdditionalAudiences, []string{"aud1", "aud2"}) {
		t.Errorf("AdditionalAudiences = %v", got.AdditionalAudiences)
	}
}

func TestClientAuthModelToClient_NullBoolsBecomeFalse(t *testing.T) {
	// Null Bool values produce zero-value bool; with `omitempty` JSON tags these
	// are then omitted server-side. Test the Go-side behavior.
	got := clientAuthModelToClient(clientAuthModel{
		Supabase:             types.BoolNull(),
		AllowTemporaryTokens: types.BoolNull(),
		AdditionalAudiences:  types.ListNull(types.StringType),
	})
	if got.Supabase {
		t.Errorf("null Bool should produce false, got Supabase=true")
	}
	if got.AllowTemporaryTokens {
		t.Errorf("null Bool should produce false, got AllowTemporaryTokens=true")
	}
}

// ── connectionsFromAPI ────────────────────────────────────────────────────────

func TestConnectionsFromAPI_NilOrEmpty(t *testing.T) {
	tests := []struct {
		name string
		in   *client.ReplicationConfig
	}{
		{"nil", nil},
		{"empty list", &client.ReplicationConfig{Connections: nil}},
		{"empty connections slice", &client.ReplicationConfig{Connections: []client.Connection{}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := connectionsFromAPI(tt.in, nil)
			if len(got) != 0 {
				t.Errorf("expected empty slice, got %d entries", len(got))
			}
			if got == nil {
				t.Errorf("expected empty slice, got nil — distinction matters for the framework")
			}
		})
	}
}

func TestConnectionsFromAPI_PreservesSecretsFromPrior(t *testing.T) {
	port5432 := int64(5432)
	apiResp := &client.ReplicationConfig{Connections: []client.Connection{{
		Type:     "postgresql",
		Name:     "main",
		Hostname: "db.example.com",
		Port:     &port5432,
		Username: "powersync_role",
		Database: "postgres",
		SSLMode:  "verify-full",
		// API never returns Password / ClientPrivateKey — these come back nil.
	}}}

	prior := []connectionModel{{
		Password:         types.StringValue("the-stored-password"),
		ClientPrivateKey: types.StringValue("the-stored-key"),
	}}

	got := connectionsFromAPI(apiResp, prior)
	if len(got) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(got))
	}
	if got[0].Password.ValueString() != "the-stored-password" {
		t.Errorf("password not preserved from prior state, got %q", got[0].Password.ValueString())
	}
	if got[0].ClientPrivateKey.ValueString() != "the-stored-key" {
		t.Errorf("client_private_key not preserved from prior state, got %q", got[0].ClientPrivateKey.ValueString())
	}
	if got[0].Hostname.ValueString() != "db.example.com" {
		t.Errorf("hostname not mapped, got %q", got[0].Hostname.ValueString())
	}
}

func TestConnectionsFromAPI_NoPriorMeansNullSecrets(t *testing.T) {
	apiResp := &client.ReplicationConfig{Connections: []client.Connection{{Type: "postgresql"}}}
	got := connectionsFromAPI(apiResp, nil)
	if !got[0].Password.IsNull() {
		t.Errorf("password should be null when no prior state, got %v", got[0].Password)
	}
	if !got[0].ClientPrivateKey.IsNull() {
		t.Errorf("client_private_key should be null when no prior state, got %v", got[0].ClientPrivateKey)
	}
}

func TestConnectionsFromAPI_NullifiesEmptyAPIFields(t *testing.T) {
	// API returns empty strings for unset fields. We map those to null in state
	// so we don't show spurious "" -> null diffs on plan.
	apiResp := &client.ReplicationConfig{Connections: []client.Connection{{
		Type:     "postgresql",
		Hostname: "db.example.com",
		// Database, SSLMode, etc. all empty strings.
	}}}
	got := connectionsFromAPI(apiResp, nil)
	if !got[0].Database.IsNull() {
		t.Errorf("empty Database should map to null, got %v", got[0].Database)
	}
	if !got[0].SSLMode.IsNull() {
		t.Errorf("empty SSLMode should map to null, got %v", got[0].SSLMode)
	}
	if !got[0].URI.IsNull() {
		t.Errorf("empty URI should map to null, got %v", got[0].URI)
	}
}

// ── clientAuthFromAPI ─────────────────────────────────────────────────────────

func TestClientAuthFromAPI_NilReturnsEmptySlice(t *testing.T) {
	got := clientAuthFromAPI(nil)
	if len(got) != 0 {
		t.Errorf("nil API config should produce empty slice, got %d entries", len(got))
	}
	if got == nil {
		t.Errorf("expected non-nil empty slice, got nil")
	}
}

func TestClientAuthFromAPI_FalseBoolsNormalizeToNull(t *testing.T) {
	// This is the "spurious diff" fix: if the API returns supabase=false and
	// allow_temporary_tokens=false, and the user never set them in HCL, we'd
	// otherwise plan a `false -> null` diff every refresh. Both should be null.
	got := clientAuthFromAPI(&client.ClientAuthConfig{
		Supabase:             false,
		AllowTemporaryTokens: false,
	})
	if len(got) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(got))
	}
	if !got[0].Supabase.IsNull() {
		t.Errorf("false Supabase should map to null, got %v", got[0].Supabase)
	}
	if !got[0].AllowTemporaryTokens.IsNull() {
		t.Errorf("false AllowTemporaryTokens should map to null, got %v", got[0].AllowTemporaryTokens)
	}
}

func TestClientAuthFromAPI_TrueBoolsKept(t *testing.T) {
	got := clientAuthFromAPI(&client.ClientAuthConfig{
		Supabase:             true,
		AllowTemporaryTokens: true,
	})
	if !got[0].Supabase.ValueBool() {
		t.Errorf("Supabase should be true")
	}
	if !got[0].AllowTemporaryTokens.ValueBool() {
		t.Errorf("AllowTemporaryTokens should be true")
	}
}

func TestClientAuthFromAPI_EmptyAudiencesNullList(t *testing.T) {
	// API returns empty/missing slice → we store null list, not empty list,
	// so unset HCL doesn't show a `[] -> null` diff.
	got := clientAuthFromAPI(&client.ClientAuthConfig{})
	if !got[0].AdditionalAudiences.IsNull() {
		t.Errorf("empty audiences should map to null list, got %v", got[0].AdditionalAudiences)
	}
}

func TestClientAuthFromAPI_AudiencesPopulated(t *testing.T) {
	got := clientAuthFromAPI(&client.ClientAuthConfig{
		AdditionalAudiences: []string{"aud1", "aud2"},
	})
	if got[0].AdditionalAudiences.IsNull() {
		t.Fatal("expected populated audiences list, got null")
	}
	var audiences []string
	diags := got[0].AdditionalAudiences.ElementsAs(context.Background(), &audiences, false)
	if diags.HasError() {
		t.Fatalf("ElementsAs failed: %v", diags)
	}
	if !reflect.DeepEqual(audiences, []string{"aud1", "aud2"}) {
		t.Errorf("audiences = %v", audiences)
	}
}

// ── buildDeployRequest ────────────────────────────────────────────────────────

func TestBuildDeployRequest_MinimalPlan(t *testing.T) {
	plan := instanceModel{
		Name:   types.StringValue("test-instance"),
		Region: types.StringValue("eu"),
	}
	got := buildDeployRequest(plan, "instance-id", "org-id", "project-id")

	if got.OrgID != "org-id" || got.AppID != "project-id" || got.ID != "instance-id" {
		t.Errorf("ID fields wrong: org=%q, app=%q, id=%q", got.OrgID, got.AppID, got.ID)
	}
	if got.Name != "test-instance" {
		t.Errorf("Name = %q", got.Name)
	}
	if got.Config.Region != "eu" {
		t.Errorf("Region = %q", got.Config.Region)
	}
	if got.ProgramVersion.Channel != "stable" {
		t.Errorf("Channel = %q, want default 'stable'", got.ProgramVersion.Channel)
	}
	if got.Config.Replication != nil {
		t.Errorf("Replication should be nil when no connections, got %+v", got.Config.Replication)
	}
	if got.Config.ClientAuth != nil {
		t.Errorf("ClientAuth should be nil when no client_auth block, got %+v", got.Config.ClientAuth)
	}
	if got.SyncRules != "" {
		t.Errorf("SyncRules should be empty when not set, got %q", got.SyncRules)
	}
}

func TestBuildDeployRequest_WithReplicationAndAuth(t *testing.T) {
	plan := instanceModel{
		Name:              types.StringValue("test"),
		Region:            types.StringValue("us"),
		SyncConfigContent: types.StringValue("config:\n  edition: 3\n"),
		ReplicationConnections: []connectionModel{{
			Type:     types.StringValue("postgresql"),
			Hostname: types.StringValue("db.example.com"),
		}},
		ClientAuth: []clientAuthModel{{
			Supabase:            types.BoolValue(true),
			AdditionalAudiences: types.ListNull(types.StringType),
		}},
	}
	got := buildDeployRequest(plan, "i", "o", "p")

	if got.SyncRules != "config:\n  edition: 3\n" {
		t.Errorf("SyncRules not threaded through, got %q", got.SyncRules)
	}
	if got.Config.Replication == nil || len(got.Config.Replication.Connections) != 1 {
		t.Fatalf("expected 1 replication connection, got %+v", got.Config.Replication)
	}
	if got.Config.Replication.Connections[0].Hostname != "db.example.com" {
		t.Errorf("connection hostname not mapped")
	}
	if got.Config.ClientAuth == nil || !got.Config.ClientAuth.Supabase {
		t.Errorf("client_auth not threaded through")
	}
}

func TestBuildDeployRequest_ProgramVersionOverride(t *testing.T) {
	plan := instanceModel{
		Name:   types.StringValue("test"),
		Region: types.StringValue("eu"),
		ProgramVersion: []programVersionModel{{
			Channel:      types.StringValue("beta"),
			VersionRange: types.StringValue("^1.2.0"),
		}},
	}
	got := buildDeployRequest(plan, "i", "o", "p")
	if got.ProgramVersion.Channel != "beta" {
		t.Errorf("Channel override not applied, got %q", got.ProgramVersion.Channel)
	}
	if got.ProgramVersion.VersionRange != "^1.2.0" {
		t.Errorf("VersionRange not applied, got %q", got.ProgramVersion.VersionRange)
	}
}

func TestBuildDeployRequest_UnknownSyncConfigOmitted(t *testing.T) {
	// During Create, SyncConfigContent might be unknown if the user didn't set it
	// (it's Computed + Optional). We must not send the string "<unknown>" to the API.
	plan := instanceModel{
		Name:              types.StringValue("test"),
		Region:            types.StringValue("eu"),
		SyncConfigContent: types.StringUnknown(),
	}
	got := buildDeployRequest(plan, "i", "o", "p")
	if got.SyncRules != "" {
		t.Errorf("unknown SyncConfigContent should not be sent, got %q", got.SyncRules)
	}
}
