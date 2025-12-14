package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlugin_IsCommand(t *testing.T) {
	p := &Plugin{}

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "valid command",
			input: "/greet",
			want:  true,
		},
		{
			name:  "command with arguments",
			input: "/greet John",
			want:  true,
		},
		{
			name:  "command with whitespace prefix",
			input: "  /greet",
			want:  true,
		},
		{
			name:  "not a command - no slash",
			input: "greet",
			want:  false,
		},
		{
			name:  "not a command - slash in middle",
			input: "hello/world",
			want:  false,
		},
		{
			name:  "empty input",
			input: "",
			want:  false,
		},
		{
			name:  "whitespace only",
			input: "   ",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.IsCommand(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseCommandInput(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantCmdName  string
		wantArgument string
	}{
		{
			name:         "command only",
			input:        "/greet",
			wantCmdName:  "greet",
			wantArgument: "",
		},
		{
			name:         "command with single argument",
			input:        "/greet John",
			wantCmdName:  "greet",
			wantArgument: "John",
		},
		{
			name:         "command with multiple arguments",
			input:        "/translate Hello World",
			wantCmdName:  "translate",
			wantArgument: "Hello World",
		},
		{
			name:         "command with extra whitespace",
			input:        "  /greet   John  ",
			wantCmdName:  "greet",
			wantArgument: "John",
		},
		{
			name:         "not a command",
			input:        "hello world",
			wantCmdName:  "",
			wantArgument: "",
		},
		{
			name:         "empty input",
			input:        "",
			wantCmdName:  "",
			wantArgument: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmdName, argument := ParseCommandInput(tt.input)
			assert.Equal(t, tt.wantCmdName, cmdName)
			assert.Equal(t, tt.wantArgument, argument)
		})
	}
}

func TestPlugin_ExpandCommand(t *testing.T) {
	// Create a plugin with test commands
	p := &Plugin{
		Commands: []Command{
			{
				Name:        "greet",
				Description: "Greet someone",
				Content:     "Say hello to $ARGUMENTS!",
			},
			{
				Name:        "simple",
				Description: "Simple command",
				Content:     "Do something simple.",
			},
		},
	}

	tests := []struct {
		name          string
		input         string
		wantErr       error
		wantSysMsg    string
		wantUserMsg   string
		wantArguments string
	}{
		{
			name:          "command with argument",
			input:         "/greet John",
			wantErr:       nil,
			wantSysMsg:    "Say hello to John!",
			wantUserMsg:   "John",
			wantArguments: "John",
		},
		{
			name:          "command without argument",
			input:         "/simple",
			wantErr:       nil,
			wantSysMsg:    "Do something simple.",
			wantUserMsg:   "",
			wantArguments: "",
		},
		{
			name:    "not a command",
			input:   "hello",
			wantErr: ErrNotACommand,
		},
		{
			name:    "unknown command",
			input:   "/unknown",
			wantErr: ErrCommandNotFound,
		},
		{
			name:          "command with multiple words as argument",
			input:         "/greet John Doe",
			wantErr:       nil,
			wantSysMsg:    "Say hello to John Doe!",
			wantUserMsg:   "John Doe",
			wantArguments: "John Doe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expanded, err := p.ExpandCommand(tt.input)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantSysMsg, expanded.SystemMessage)
			assert.Equal(t, tt.wantUserMsg, expanded.UserMessage)
			assert.Equal(t, tt.wantArguments, expanded.Arguments)
		})
	}
}

func TestPlugin_ProcessInput(t *testing.T) {
	p := &Plugin{
		Commands: []Command{
			{
				Name:    "greet",
				Content: "Say hello to $ARGUMENTS!",
			},
		},
	}

	tests := []struct {
		name        string
		input       string
		wantOpt     bool
		wantUserMsg string
		wantErr     bool
	}{
		{
			name:        "valid command",
			input:       "/greet John",
			wantOpt:     true,
			wantUserMsg: "John",
			wantErr:     false,
		},
		{
			name:        "not a command - returns original input",
			input:       "hello world",
			wantOpt:     false,
			wantUserMsg: "hello world",
			wantErr:     false,
		},
		{
			name:        "unknown command - returns error",
			input:       "/unknown",
			wantOpt:     false,
			wantUserMsg: "/unknown",
			wantErr:     true,
		},
		{
			name:        "command without arguments - uses original input",
			input:       "/greet",
			wantOpt:     true,
			wantUserMsg: "/greet",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt, userMsg, err := p.ProcessInput(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.wantOpt {
				assert.NotNil(t, opt)
			} else {
				assert.Nil(t, opt)
			}
			assert.Equal(t, tt.wantUserMsg, userMsg)
		})
	}
}

func TestCommand_ToOption(t *testing.T) {
	cmd := &Command{
		Name:    "test",
		Content: "Test content for system message.",
	}

	opt := cmd.ToOption()
	assert.NotNil(t, opt)
}

func TestCommand_ToOptionWithArgs(t *testing.T) {
	cmd := &Command{
		Name:    "greet",
		Content: "Hello, $ARGUMENTS! How are you?",
	}

	tests := []struct {
		name string
		args string
	}{
		{
			name: "with argument",
			args: "John",
		},
		{
			name: "empty argument",
			args: "",
		},
		{
			name: "multiple word argument",
			args: "John Doe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := cmd.ToOptionWithArgs(tt.args)
			assert.NotNil(t, opt)
		})
	}
}

func TestExpandedCommand_ToOption(t *testing.T) {
	expanded := &ExpandedCommand{
		SystemMessage: "Expanded system message",
		UserMessage:   "user input",
		Arguments:     "user input",
	}

	opt := expanded.ToOption()
	assert.NotNil(t, opt)
}

func TestErrNotACommand(t *testing.T) {
	assert.NotNil(t, ErrNotACommand)
	assert.Contains(t, ErrNotACommand.Error(), "not a slash command")
}

func TestErrCommandNotFound(t *testing.T) {
	assert.NotNil(t, ErrCommandNotFound)
	assert.Contains(t, ErrCommandNotFound.Error(), "not found")
}
