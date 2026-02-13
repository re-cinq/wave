package adapter

import "testing"

func TestShelljoinArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "simple flags",
			args: []string{"-p", "--model", "opus"},
			want: "-p --model opus",
		},
		{
			name: "prompt with spaces",
			args: []string{"-p", "hello world"},
			want: "-p 'hello world'",
		},
		{
			name: "prompt with pipe and ampersand",
			args: []string{"-p", "test | cmd & bg"},
			want: `-p 'test | cmd & bg'`,
		},
		{
			name: "prompt with single quotes",
			args: []string{"-p", "it's a test"},
			want: `-p 'it'\''s a test'`,
		},
		{
			name: "prompt with dollar sign",
			args: []string{"-p", "echo $HOME"},
			want: `-p 'echo $HOME'`,
		},
		{
			name: "empty argument",
			args: []string{"-p", ""},
			want: "-p ''",
		},
		{
			name: "complex mixed args",
			args: []string{"--model", "opus", "--allowedTools", "Read,Write,Edit,Bash", "Say: hello & goodbye"},
			want: "--model opus --allowedTools Read,Write,Edit,Bash 'Say: hello & goodbye'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shelljoinArgs(tt.args)
			if got != tt.want {
				t.Errorf("shelljoinArgs(%v)\n  got  %q\n  want %q", tt.args, got, tt.want)
			}
		})
	}
}
