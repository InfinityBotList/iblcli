package helpers

import "fmt"

func GetUsername(userId string) (string, error) {
	resp, err := NewReq().Get("users/" + userId).Do()

	if err != nil {
		return "", err
	}

	if resp.Response.StatusCode != 200 {
		return "", fmt.Errorf("error getting username, got status code %d", resp.Response.StatusCode)
	}

	var user struct {
		User struct {
			Username string `json:"username"`
		} `json:"user"`
	}

	err = resp.Json(&user)

	if err != nil {
		return "", err
	}

	return user.User.Username, nil
}
