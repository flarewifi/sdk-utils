package main

import tools "core/utils"

func main() {
	name, err := tools.AskCmdInput("Enter migration name, e.g. create_users_table")
	if err != nil {
		panic(err)
	}
	tools.MigrationCreate("core", name)
}
