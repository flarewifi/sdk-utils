package shell

func ExecAll(commands []string) error {
	for _, c := range commands {
		err := Exec(c, nil)
		if err != nil {
			return err
		}
	}
	return nil
}
