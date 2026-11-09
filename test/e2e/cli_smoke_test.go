//go:build e2e

package e2e

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	fixtureNamespace    = "migration-e2e-ecr-secret"
	fixtureSubscription = "ecr-secret-operator"
	fixtureCatalog      = "operatorhubio-catalog"
)

type cliRecord struct {
	Type   string          `json:"type"`
	Target string          `json:"target"`
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
}

// TestMigrationCLISmoke follows the catalog, check, preview, convert, and
// rollback path from library-olm's fixture E2E suite using this repo's binaries.
func TestMigrationCLISmoke(t *testing.T) {
	kubeconfig := os.Getenv("E2E_KUBECONFIG")
	clusterName := os.Getenv("E2E_CLUSTER_NAME")
	if kubeconfig == "" || clusterName == "" {
		t.Fatal("E2E_KUBECONFIG and E2E_CLUSTER_NAME are required")
	}
	if _, err := os.Stat(kubeconfig); err != nil {
		t.Fatalf("E2E_KUBECONFIG: %v", err)
	}
	kubectl := func(args ...string) string {
		t.Helper()
		return run(t, "kubectl", append([]string{"--kubeconfig", kubeconfig}, args...)...)
	}
	if context := strings.TrimSpace(kubectl("config", "current-context")); context != "kind-"+clusterName {
		t.Fatalf("refusing to mutate context %q; want dedicated kind cluster %q", context, "kind-"+clusterName)
	}
	if controller := strings.TrimSpace(kubectl("get", "deployment/olm-operator", "-n", "olm", "--ignore-not-found", "-o", "name")); controller != "" {
		t.Fatalf("fixture cluster unexpectedly has an OLMv0 controller: %s", controller)
	}
	if existing := strings.TrimSpace(kubectl("get", "clustercatalog/"+fixtureCatalog, "--ignore-not-found", "-o", "name")); existing != "" {
		t.Fatalf("fixture catalog already exists: %s", existing)
	}
	if existing := strings.TrimSpace(kubectl("get", "clusterextension/"+fixtureSubscription, "--ignore-not-found", "-o", "name")); existing != "" {
		t.Fatalf("fixture operator is already migrated: %s", existing)
	}
	kubectl("get", "subscription/"+fixtureSubscription, "-n", fixtureNamespace)
	kubectl("get", "catalogsource/"+fixtureCatalog, "-n", "olm")
	deployment := "deployment/" + fixtureSubscription + "-controller-manager"
	workloadUID := strings.TrimSpace(kubectl("get", deployment, "-n", fixtureNamespace, "-o", "jsonpath={.metadata.uid}"))
	if workloadUID == "" {
		t.Fatal("fixture operator Deployment has no UID")
	}

	catalogCLI := binary(t, "migrate-catalogs-v0-to-v1")
	operatorCLI := binary(t, "migrate-operators-v0-to-v1")
	catalogTarget := "olm/" + fixtureCatalog
	operatorTarget := fixtureNamespace + "/" + fixtureSubscription

	preview := run(t, catalogCLI, "--kubeconfig", kubeconfig, "--dry-run", "--output=jsonl")
	assertCatalogOutcome(t, requireRecord(t, preview, "result", catalogTarget), "dry-run")
	if existing := strings.TrimSpace(kubectl("get", "clustercatalog/"+fixtureCatalog, "--ignore-not-found", "-o", "name")); existing != "" {
		t.Fatal("catalog dry-run created a ClusterCatalog")
	}

	created := run(t, catalogCLI, "--kubeconfig", kubeconfig, "--output=jsonl")
	assertCatalogOutcome(t, requireRecord(t, created, "result", catalogTarget), "created")
	kubectl("wait", "--for=jsonpath={.status.conditions[?(@.type=='Serving')].status}=True", "clustercatalog/"+fixtureCatalog, "--timeout=10m")

	check := run(t, operatorCLI, "check", fixtureSubscription, "-n", fixtureNamespace, "--kubeconfig", kubeconfig, "--output=jsonl")
	var checkData struct {
		Status string `json:"status"`
	}
	decodeData(t, requireRecord(t, check, "check", operatorTarget), &checkData)
	if checkData.Status != "Eligible" {
		t.Fatalf("operator check status = %q, want Eligible:\n%s", checkData.Status, check)
	}

	operatorPreview := run(t, operatorCLI, "convert", fixtureSubscription, "-n", fixtureNamespace, "--dry-run", "--kubeconfig", kubeconfig, "--output=jsonl")
	var previewData struct {
		ClusterObjectSet        string         `json:"cluster_object_set"`
		ClusterObjectDeployment string         `json:"cluster_object_deployment"`
		KindCounts              map[string]int `json:"kind_counts"`
	}
	decodeData(t, requireRecord(t, operatorPreview, "dry-run", operatorTarget), &previewData)
	if previewData.ClusterObjectSet == "" && previewData.ClusterObjectDeployment == "" {
		t.Fatalf("operator dry-run omitted its resource manager:\n%s", operatorPreview)
	}
	if len(previewData.KindCounts) == 0 {
		t.Fatalf("operator dry-run omitted collected resource kinds:\n%s", operatorPreview)
	}
	if existing := strings.TrimSpace(kubectl("get", "clusterextension/"+fixtureSubscription, "--ignore-not-found", "-o", "name")); existing != "" {
		t.Fatal("operator dry-run created a ClusterExtension")
	}
	kubectl("get", "subscription/"+fixtureSubscription, "-n", fixtureNamespace)

	converted := run(t, operatorCLI, "convert", fixtureSubscription, "-n", fixtureNamespace, "--kubeconfig", kubeconfig, "--output=jsonl")
	if record := requireRecord(t, converted, "result", operatorTarget); record.Status != "completed" {
		t.Fatalf("conversion status = %q, want completed:\n%s", record.Status, converted)
	}
	kubectl("wait", "--for=jsonpath={.status.conditions[?(@.type=='Installed')].status}=True", "clusterextension/"+fixtureSubscription, "--timeout=10m")
	if remaining := strings.TrimSpace(kubectl("get", "subscription/"+fixtureSubscription, "-n", fixtureNamespace, "--ignore-not-found", "-o", "name")); remaining != "" {
		t.Fatalf("conversion retained Subscription: %s", remaining)
	}
	if uid := strings.TrimSpace(kubectl("get", deployment, "-n", fixtureNamespace, "-o", "jsonpath={.metadata.uid}")); uid != workloadUID {
		t.Fatalf("conversion changed workload UID from %s to %s", workloadUID, uid)
	}

	rolledBack := run(t, operatorCLI, "rollback", fixtureSubscription, "--acknowledge-installed", "--kubeconfig", kubeconfig, "--output=jsonl")
	if record := requireRecord(t, rolledBack, "result", fixtureSubscription); record.Status != "completed" {
		t.Fatalf("rollback status = %q, want completed:\n%s", record.Status, rolledBack)
	}
	kubectl("get", "subscription/"+fixtureSubscription, "-n", fixtureNamespace)
	if remaining := strings.TrimSpace(kubectl("get", "clusterextension/"+fixtureSubscription, "--ignore-not-found", "-o", "name")); remaining != "" {
		t.Fatalf("rollback retained ClusterExtension: %s", remaining)
	}
	if uid := strings.TrimSpace(kubectl("get", deployment, "-n", fixtureNamespace, "-o", "jsonpath={.metadata.uid}")); uid != workloadUID {
		t.Fatalf("rollback changed workload UID from %s to %s", workloadUID, uid)
	}
}

func binary(t *testing.T, name string) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "..", "bin", name))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("migration binary %s: %v", path, err)
	}
	return path
}

func run(t *testing.T, name string, args ...string) string {
	t.Helper()
	command := exec.CommandContext(t.Context(), name, args...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(args, " "), err, output)
	}
	return string(output)
}

func requireRecord(t *testing.T, stream, recordType, target string) cliRecord {
	t.Helper()
	for _, line := range strings.Split(strings.TrimSpace(stream), "\n") {
		var record cliRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("invalid JSONL record: %v\n%s", err, stream)
		}
		if record.Type == recordType && record.Target == target {
			return record
		}
	}
	t.Fatalf("missing %s record for %s:\n%s", recordType, target, stream)
	return cliRecord{}
}

func decodeData(t *testing.T, record cliRecord, value any) {
	t.Helper()
	if err := json.Unmarshal(record.Data, value); err != nil {
		t.Fatalf("decode %s record for %s: %v", record.Type, record.Target, err)
	}
}

func assertCatalogOutcome(t *testing.T, record cliRecord, want string) {
	t.Helper()
	var data struct {
		Outcome string `json:"outcome"`
	}
	decodeData(t, record, &data)
	if record.Status != "completed" || data.Outcome != want {
		t.Fatalf("catalog result = %s/%s, want completed/%s", record.Status, data.Outcome, want)
	}
}
