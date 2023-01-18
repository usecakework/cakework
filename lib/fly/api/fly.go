package api

import "github.com/usecakework/cakework/lib/auth"



type Fly struct {
	AccessToken string
	Org string
	CredentialsProvider auth.BearerCredentialsProvider
}

func New(accessToken string, org string, credentialsProvider auth.BearerCredentialsProvider) *Fly {
	fly := &Fly{
		AccessToken: accessToken,
		Org: org,
		CredentialsProvider: credentialsProvider,
	}

	return fly
}

// Q: should this return machine info?
func (fly *Fly) NewMachine(userId string, project string, task string, flyApp string, name string, image string) error {
	// make a post request to the internal fly api endpoint

	return nil
}