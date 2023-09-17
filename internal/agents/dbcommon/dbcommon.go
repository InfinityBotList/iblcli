package dbcommon

import "os"

// Creates a env based on os.Environ()
func CreateEnv() []string {
	var env []string = make([]string, 0)
	if os.Getenv("PGDATABASE") != "" {
		env = append(env, "PGDATABASE="+os.Getenv("PGDATABASE"))
	} else {
		env = append(env, "PGDATABASE=infinity")
	}

	if os.Getenv("PGUSER") != "" {
		env = append(env, "PGUSER="+os.Getenv("PGUSER"))
	}

	return env
}
