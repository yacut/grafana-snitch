package main

import (
	"context"
	"io/ioutil"

	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2/google"
	admin "google.golang.org/api/admin/directory/v1"
)

// Build and returns an Admin SDK Directory service object authorized with
// the service accounts that act on behalf of the given user.
// Args:
//    googleAdminConfigPath: The Path to the Service Account's Private Key file
//    googleAdminEmail: The email of the user. Needs permissions to access the Admin APIs.
// Returns:
//    Admin SDK directory service object.
func getService(googleAdminConfigPath string, googleAdminEmail string) *admin.Service {
	jsonCredentials, err := ioutil.ReadFile(googleAdminConfigPath)
	if err != nil {
		promErrors.WithLabelValues("get-admin-config").Inc()
		log.Fatal().Err(err).Msg("Unable to read client secret file.")
		return nil
	}

	config, err := google.JWTConfigFromJSON(jsonCredentials, admin.AdminDirectoryGroupMemberReadonlyScope, admin.AdminDirectoryGroupReadonlyScope)
	if err != nil {
		promErrors.WithLabelValues("get-admin-config").Inc()
		log.Fatal().Err(err).Msg("Unable to parse client secret file to config.")
		return nil
	}
	config.Subject = googleAdminEmail
	ctx := context.Background()
	client := config.Client(ctx)

	srv, err := admin.New(client)
	if err != nil {
		promErrors.WithLabelValues("get-admin-client").Inc()
		log.Fatal().Err(err).Msg("Unable to retrieve Group Settings Client.")
		return nil
	}
	return srv
}

// Gets recursive the group members by email and returns the user list
// Args:
//    service: Admin SDK directory service object.
//    email: The email of the group.
// Returns:
//    Admin SDK member list.
func getMembers(service *admin.Service, email string) ([]*admin.Member, error) {
	result, err := service.Members.List(email).Do()
	if err != nil {
		promErrors.WithLabelValues("get-members").Inc()
		log.Fatal().Err(err).Msg("Unable to get group members.")
		return nil, err
	}

	var userList []*admin.Member
	for _, member := range result.Members {
		if member.Type == "GROUP" {
			groupMembers, _ := getMembers(service, member.Email)
			userList = append(userList, groupMembers...)
		} else {
			userList = append(userList, member)
		}
	}

	return userList, nil
}
