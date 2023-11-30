// Copyright 2023 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"context"
	"fmt"

	resourcemanager "cloud.google.com/go/resourcemanager/apiv3"
	"github.com/posener/complete/v2/predict"
	"github.com/sqin2019/access-on-demand-fork/apis/v1alpha1"
	"github.com/sqin2019/access-on-demand-fork/pkg/handler"
	"github.com/sqin2019/access-on-demand-fork/pkg/requestutil"

	"github.com/abcxyz/pkg/cli"
)

var _ cli.Command = (*IAMCleanupCommand)(nil)

// IAMCleanupCommand handles IAM requests.
type IAMCleanupCommand struct {
	cli.BaseCommand

	flagPath string

	// flagExpiry time.Time

	// testHandler is used for testing only.
	testHandler iamHandler
}

func (c *IAMCleanupCommand) Desc() string {
	return `Handle the IAM request YAML file in the given path`
}

func (c *IAMCleanupCommand) Help() string {
	return `
Usage: {{ COMMAND }} [options]

Handle the IAM request YAML file in the given path:

      {{ COMMAND }} -path "/path/to/file.yaml" -duration "2h" -start-time "2009-11-10T23:00:00Z"
`
}

func (c *IAMCleanupCommand) Flags() *cli.FlagSet {
	set := cli.NewFlagSet()

	// Command options
	f := set.NewSection("COMMAND OPTIONS")

	f.StringVar(&cli.StringVar{
		Name:    "path",
		Target:  &c.flagPath,
		Example: "/path/to/file.yaml",
		Predict: predict.Files("*"),
		Usage:   `The path of IAM request file, in YAML format.`,
	})

	// f.TimeVar(time.RFC3339, &cli.TimeVar{
	// 	Name:    "expiry",
	// 	Target:  &c.flagExpiry,
	// 	Example: "2009-11-10T23:00:00Z",
	// 	Usage: `The expiry time of the IAM permission lifecycle in RFC3339 format. `,
	// })
	return set
}

func (c *IAMCleanupCommand) Run(ctx context.Context, args []string) error {
	f := c.Flags()
	if err := f.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}
	args = f.Args()
	if len(args) > 0 {
		return fmt.Errorf("unexpected arguments: %q", args)
	}

	if c.flagPath == "" {
		return fmt.Errorf("path is required")
	}

	// if c.flagExpiry.Equal(time.Time{}) {
	// 	return fmt.Errorf("expiry is required")
	// }

	return c.handleIAM(ctx)
}

func (c *IAMCleanupCommand) handleIAM(ctx context.Context) error {
	// Read request from file path.
	var req v1alpha1.IAMRequest
	if err := requestutil.ReadRequestFromPath(c.flagPath, &req); err != nil {
		return fmt.Errorf("failed to read %T: %w", &req, err)
	}

	if err := v1alpha1.ValidateIAMRequest(&req); err != nil {
		return fmt.Errorf("failed to validate %T: %w", &req, err)
	}

	var h iamHandler
	if c.testHandler != nil {
		// Use testHandler if it is for testing.
		h = c.testHandler
	} else {
		// Create resource manager clients.
		organizationsClient, err := resourcemanager.NewOrganizationsClient(ctx)
		if err != nil {
			return fmt.Errorf("failed to create organizations client: %w", err)
		}
		defer organizationsClient.Close()

		foldersClient, err := resourcemanager.NewFoldersClient(ctx)
		if err != nil {
			return fmt.Errorf("failed to create folders client: %w", err)
		}
		defer foldersClient.Close()

		projectsClient, err := resourcemanager.NewProjectsClient(ctx)
		if err != nil {
			return fmt.Errorf("failed to create projects client: %w", err)
		}
		defer projectsClient.Close()

		// Create IAMHandler with the clients.
		h, err = handler.NewIAMHandler(
			ctx,
			organizationsClient,
			foldersClient,
			projectsClient,
		)
		if err != nil {
			return fmt.Errorf("failed to create IAM handler: %w", err)
		}
	}

	// Wrap IAMRequest to include Duration.
	reqWrapper := &v1alpha1.IAMRequestWrapper{
		IAMRequest: &req,
	}
	// TODO(#15): add a log level to output handler response.
	if _, err := h.Cleanup(ctx, reqWrapper); err != nil {
		return fmt.Errorf("failed to clean up IAM request: %w", err)
	}
	c.Outf("Successfully Cleanuped IAM request")

	return nil
}
