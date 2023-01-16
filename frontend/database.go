package main

// TODO put into a separate package. Can have the main.go invoke this as well
func getUserFromAPIKey(apiKey string) (*User, error) {
	// fetch the client token by the token value
	// return the user
	newRequest := GetUserByClientTokenRequest {
		Token: apiKey,
	}
	// TODO: before calling the db, we need to generate additional fields like the status and request id. so bind to a new object?

	var user User
	err = db.QueryRow("SELECT userId FROM ClientToken where token = ?", newRequest.Token).Scan(&user.Id)
	if err != nil && user.Id != "" {
		return nil, err
	} else {
		return &user, nil
	}
}
