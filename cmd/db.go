package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/InfinityBotList/ibl/internal/agents/dbcommon"
	"github.com/InfinityBotList/ibl/internal/devmode"
	"github.com/InfinityBotList/ibl/types"
	"github.com/spf13/cobra"
)

// copyDb represents the copydb command
var copyDb = &cobra.Command{
	Use:   "copydb TO",
	Short: "Copies the database from 'olympia' to current server. User must currently be on 'olympia'",
	Long:  `Add an experiment to a user`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		defer func() {
			fmt.Println("Cleaning up...")

			// delete all files in work directory
			err := os.RemoveAll("work")

			if err != nil {
				fmt.Println("Error cleaning up:", err)
			}
		}()

		// create a work directory
		err := os.Mkdir("work", 0755)

		if err != nil {
			fmt.Println("Error creating work directory:", err)
			return
		}

		fmt.Println("Creating database backup as schema.sql")

		backupCmd := exec.Command("pg_dump", "-Fc", "-d", "infinity", "-f", "work/schema.sql")

		backupCmd.Env = dbcommon.CreateEnv()

		err = backupCmd.Run()

		if err != nil {
			fmt.Println("Error when creating db backup", err)
			return
		}

		fmt.Println("Copying file to target server")

		cpCmd := exec.Command("scp", "work/schema.sql", fmt.Sprintf("root@%s:/tmp/schema.sql", args[0]))
		cpCmd.Env = os.Environ()
		err = cpCmd.Run()

		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println("Restoring database on target server")

		cmds := [][]string{
			/*{
				"psql", "-c", "'DROP ROLE IF EXISTS enfinity'",
			},
			{
				"psql", "-c", "'DROP ROLE IF EXISTS infinity'",
			},
			{
				"psql", "-c", "'CREATE ROLE enfinity'",
			},
			{
				"psql", "-c", "'ALTER ROLE enfinity WITH LOGIN'",
			},
			{
				"psql", "-c", "'CREATE ROLE infinity'",
			},
			{
				"psql", "-c", "'ALTER ROLE infinity WITH LOGIN'",
			},*/
			{
				"psql", "-c", "'DROP DATABASE IF EXISTS infinity_bak'",
			},
			{
				"psql", "-c", "'CREATE DATABASE infinity_bak'",
			},
			{
				"psql", "-d", "infinity_bak", "-c", "'CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"'",
			},
			{
				"psql", "-d", "infinity_bak", "-c", "'CREATE EXTENSION IF NOT EXISTS \"citext\"'",
			},
			{
				"pg_restore", "-d", "infinity_bak", "/tmp/schema.sql",
			},
			{
				"psql", "-d", "infinity_bak", "-c", "'UPDATE webhooks SET secret = uuid_generate_v4()::text'",
			},
			{
				"psql", "-d", "infinity_bak", "-c", "'UPDATE users SET api_token = uuid_generate_v4()::text'",
			},
			{
				"psql", "-d", "infinity_bak", "-c", "'UPDATE bots SET api_token = uuid_generate_v4()::text'",
			},
			{
				"psql", "-d", "infinity_bak", "-c", "'UPDATE servers SET api_token = uuid_generate_v4()::text'",
			},
			{
				"psql", "-d", "infinity", "-c", "'DROP DATABASE IF EXISTS infinity_old'",
			},
			{
				"psql", "-d", "infinity_bak", "-c", "'ALTER DATABASE infinity RENAME TO infinity_old'",
			},
			{
				"psql", "-c", "'ALTER DATABASE infinity_bak RENAME TO infinity'",
			},
			{
				"psql", "-d", "infinity", "-c", "'DROP DATABASE IF EXISTS infinity_old'",
			},
		}

		for _, c := range cmds {
			fmt.Println("=>", strings.Join(c, " "))

			cmd := exec.Command("ssh", args[0], strings.Join(c, " "))

			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Env = os.Environ()

			err = cmd.Run()

			if err != nil {
				fmt.Println(err)
				return
			}
		}
	},
}

// dbCmd represents the db command
var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "DB operations",
	Long:  `DB operations`,
}

func init() {
	if devmode.DevMode().Allows(types.DevModeFull) {
		dbCmd.AddCommand(copyDb)
		rootCmd.AddCommand(dbCmd)
	}
}
