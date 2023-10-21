package cmd

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
)

// addExpCommand represents the add experiment command
var addExpCommand = &cobra.Command{
	Use:     "addexp USERID EXPERIMENT",
	Short:   "Add an experiment to a user",
	Long:    `Add an experiment to a user`,
	Args:    cobra.ExactArgs(2),
	Aliases: []string{"addexperiment", "ae"},
	Run: func(cmd *cobra.Command, args []string) {
		pool, err := pgxpool.Connect(context.Background(), "postgres:///infinity")

		if err != nil {
			panic(err)
		}

		// Check experiments from db
		var experiments []string

		err = pool.QueryRow(context.Background(), "SELECT experiments FROM users WHERE user_id = $1", args[0]).Scan(&experiments)

		if err != nil {
			panic(err)
		}

		if slices.Contains(experiments, args[1]) {
			fmt.Println("User already has experiment")
			return
		}

		experiments = append(experiments, args[1])

		_, err = pool.Exec(context.Background(), "UPDATE users SET experiments = $1 WHERE user_id = $2", experiments, args[0])

		if err != nil {
			panic(err)
		}
	},
}

var remExpCommand = &cobra.Command{
	Use:     "remexp USERID EXPERIMENT",
	Short:   "Remove an experiment from a user",
	Long:    `Remove an experiment from a user`,
	Args:    cobra.ExactArgs(2),
	Aliases: []string{"removeexperiment", "re", "delexp"},
	Run: func(cmd *cobra.Command, args []string) {
		pool, err := pgxpool.Connect(context.Background(), "postgres:///infinity")

		if err != nil {
			panic(err)
		}

		// Check experiments from db
		var experiments []string

		err = pool.QueryRow(context.Background(), "SELECT experiments FROM users WHERE user_id = $1", args[0]).Scan(&experiments)

		if err != nil {
			panic(err)
		}

		if !slices.Contains(experiments, args[1]) {
			fmt.Println("User doesn't have experiment")
			return
		}

		var newExperiments = []string{}

		for _, experiment := range experiments {
			if experiment != args[1] {
				newExperiments = append(newExperiments, experiment)
			}
		}

		_, err = pool.Exec(context.Background(), "UPDATE users SET experiments = $1 WHERE user_id = $2", newExperiments, args[0])

		if err != nil {
			panic(err)
		}
	},
}

// adminCmd represents the admin command
var adminCmd = &cobra.Command{
	Use:   "admin",
	Short: "Admin operations",
	Long:  `Admin operations`,
}

func init() {
	adminCmd.AddCommand(addExpCommand)
	adminCmd.AddCommand(remExpCommand)
	rootCmd.AddCommand(adminCmd)
}
