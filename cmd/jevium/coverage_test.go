package main

import "testing"

func TestAuditURLsRequireReportBeforeStartup(t *testing.T) {
	t.Setenv("TYPESAFE_API_KEY", "")
	err := run([]string{"--mode", "chrome", "--url", "https://example.test", "--goal", "inspect", "--audit-url", "https://example.test"})
	if err == nil || err.Error() != "--audit-url requires --coverage" {
		t.Fatalf("validation error=%v", err)
	}
}
