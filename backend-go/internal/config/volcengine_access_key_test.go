package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVolcengineAccessKeySurvivesAccountSyncAndReload(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	data := `{
  "managedAccounts":[{"accountUid":"acct_volc","providerId":"volcengine","name":"volc","credentials":[{"credentialUid":"cred_volc","apiKey":"ark-inference"}]}],
  "upstream":[{"accountUid":"acct_volc","channelUid":"ch_volc","providerId":"volcengine","name":"volc-claude","serviceType":"claude","autoManaged":true,"baseUrl":"https://ark.cn-beijing.volces.com/api/plan","apiKeyConfigs":[{"credentialUid":"cred_volc","baseUrl":"https://ark.cn-beijing.volces.com/api/plan"}]}],
  "chatUpstream":[],"responsesUpstream":[],"geminiUpstream":[],"imagesUpstream":[],"vectorsUpstream":[]
}`
	if err := os.WriteFile(configPath, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	manager, err := NewConfigManager(configPath, filepath.Join(dir, "backups"))
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.SetManagedAccountVolcengineAccessKey("acct_volc", "cred_volc", "AKID", "SECRET"); err != nil {
		t.Fatal(err)
	}
	if err := manager.SetManagedAccountVolcenginePlan("acct_volc", "cred_volc", "agent_plan", "Large", "Running"); err != nil {
		t.Fatal(err)
	}
	usedPercent := 12.5
	if err := manager.SetManagedAccountVolcenginePlanUsage("acct_volc", "cred_volc", &VolcenginePlanUsage{
		FiveHour: &VolcenginePlanUsageWindow{UsedPercent: &usedPercent},
	}); err != nil {
		t.Fatal(err)
	}
	if err := manager.RenameManagedAccount("acct_volc", "volc-renamed"); err != nil {
		t.Fatal(err)
	}
	_ = manager.Close()

	reloaded, err := NewConfigManager(configPath, filepath.Join(dir, "backups-reload"))
	if err != nil {
		t.Fatal(err)
	}
	defer reloaded.Close()
	credential, ok := reloaded.GetManagedAccountCredential("acct_volc", "cred_volc")
	if !ok || credential.VolcengineAccessKey == nil {
		t.Fatalf("AK/SK 未持久化: %+v", credential)
	}
	if pair := credential.VolcengineAccessKey; pair.AccessKeyID != "AKID" || pair.SecretAccessKey != "SECRET" || pair.Plan != "agent_plan" || pair.PlanTier != "Large" {
		t.Fatalf("持久化内容不匹配: %+v", pair)
	}
	usage := credential.VolcengineAccessKey.Usage
	if usage == nil || usage.FiveHour == nil || usage.FiveHour.UsedPercent == nil || *usage.FiveHour.UsedPercent != usedPercent {
		t.Fatalf("用量百分比未持久化: %+v", usage)
	}
}
