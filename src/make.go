package pike

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/rs/zerolog/log"
)

// Make creates the required role
func Make(directory string) (*string, error) {
	err := Scan(
		directory,
		"terraform",
		nil,
		true,
		true,
	)
	if err != nil {
		return nil, err
	}

	directory, err = filepath.Abs(directory)
	if err != nil {
		return nil, fmt.Errorf("failed to find path %w", err)
	}

	policyPath, err := filepath.Abs(directory + "/.pike/")
	if err != nil {
		return nil, err
	}

	tf, err2 := tfApply(policyPath)
	if err2 != nil {
		return nil, err2
	}

	state, err := tf.Show(context.Background())
	if err != nil {
		return nil, err
	}

	if (state.Values.Outputs["arn"]) != nil {
		arn := state.Values.Outputs["arn"]
		log.Printf("aws role create/updated %s", arn.Value.(string))
		role := arn.Value.(string)

		return &role, nil
	}

	return nil, errors.New("no arn found in state")
}

func tfApply(policyPath string) (*tfexec.Terraform, error) {
	tfPath, err := LocateTerraform()
	if err != nil {
		return nil, err
	}

	terraform, err := tfexec.NewTerraform(policyPath, tfPath)
	if err != nil {
		return nil, err
	}

	err = terraform.Init(context.Background(), tfexec.Upgrade(true))
	if err != nil {
		return nil, err
	}

	err = terraform.Apply(context.Background())
	if err != nil {
		return nil, fmt.Errorf("terraform apply failed %w", err)
	}

	return terraform, nil
}

// Apply  executes tf using prepared role
func Apply(target string, region string) error {
	iamRole, err := Make(target)
	time.Sleep(5 * time.Second)

	if err != nil {
		return err
	}
	// clear any temp creds
	unSetAWSAuth()

	err = setAWSAuth(*iamRole, region)
	if err != nil {
		unSetAWSAuth()

		return err
	}

	_, err = tfApply(target)

	if err == nil {
		log.Printf("provisioned %s", target)
	}

	unSetAWSAuth()

	return err
}
