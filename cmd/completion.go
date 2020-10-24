package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(completionCmd)
}

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh]",
	Short: "Generate completion script",
	Long: `To load completions:

Bash:

$ source <(ddbt completion bash)

# To load completions for each session, execute once:
Linux:
  $ ddbt completion bash > /etc/bash_completion.d/ddbt
MacOS:
  $ ddbt completion bash > /usr/local/etc/bash_completion.d/ddbt

Zsh:

# If shell completion is not already enabled in your environment you will need
# to enable it.  You can execute the following once:

$ echo "autoload -U compinit; compinit" >> ~/.zshrc

# To load completions for each session, execute once:
$ ddbt completion zsh > "${fpath[1]}/_ddbt"

# You will need to start a new shell for this setup to take effect.`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh"},
	Args:                  cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		}
	},
}
