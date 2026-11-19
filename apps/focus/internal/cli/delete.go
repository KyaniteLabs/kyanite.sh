package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/kyanite/focus/internal/engine"
	"github.com/kyanite/focus/internal/repository"
	"github.com/kyanite/focus/pkg/utils"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "remove <task-id>",
	Short: "🗑️ Erase a mission from the digital grid",
	Long:  "🌸 Permanently remove tasks from your digital realm",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Initialize components
		repo := repository.NewStoreRepository(utils.GetStoragePath())
		eng := engine.New(repo)

		runDeleteTask(eng, args[0], os.Stdin, os.Stdout)
	},
}

// runDeleteTask deletes one task after showing it and asking a y/N
// confirmation. Empty input or anything other than y/yes cancels the
// delete before the store is touched.
func runDeleteTask(eng *engine.Engine, taskID string, in io.Reader, out io.Writer) {
	fmt.Fprintln(out, "🌪️ Initializing Mission Erasure Protocol 🌪️")
	fmt.Fprintln(out, "~-~-~-~-~-~-~-~-~-~-~-~-~-~-~-~-~-~-~-~-~-~-~-~-~-~-~")

	// Get task first to show what we're deleting
	task, err := eng.GetTask(taskID)
	if err != nil {
		fmt.Fprintf(out, "❌ Mission not found: %v\n", err)
		return
	}

	fmt.Fprintf(out, "📋 %s\n", task.Description)
	fmt.Fprintf(out, "🔢 ID: %s\n", task.ID)

	if !confirmYesNo(in, out, "⚠️  Erase this mission permanently") {
		fmt.Fprintln(out, "❌ Mission erasure cancelled.")
		return
	}

	// Delete task
	err = eng.DeleteTask(taskID)
	if err != nil {
		fmt.Fprintf(out, "❌ Error erasing mission: %v\n", err)
		return
	}

	fmt.Fprintln(out, "✅ Mission erased successfully!")
}
