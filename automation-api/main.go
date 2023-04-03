package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/spf13/cobra"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

var (
	WarningLogger *log.Logger
	InfoLogger    *log.Logger
	ErrorLogger   *log.Logger
	DebugLogger   *log.Logger
)

func init() {
	WarningLogger = log.New(os.Stdout, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	InfoLogger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	DebugLogger = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func createMincraftServer(ctx *pulumi.Context) error {
	vpc, err := ec2.NewVpc(ctx, "minecraft-vpc", &ec2.VpcArgs{
		CidrBlock:          pulumi.String("10.0.0.0/16"),
		EnableDnsHostnames: pulumi.Bool(true),
		EnableDnsSupport:   pulumi.Bool(true),
	})
	if err != nil {
		return err
	}
	subnet, err := ec2.NewSubnet(ctx, "minecraft-subnet", &ec2.SubnetArgs{
		VpcId:               vpc.ID(),
		CidrBlock:           pulumi.String("10.0.48.0/20"),
		MapPublicIpOnLaunch: pulumi.Bool(true),
		AvailabilityZone:    pulumi.String("eu-central-1a"),
	})
	if err != nil {
		return err
	}
	internetGateway, err := ec2.NewInternetGateway(ctx, "minecraft-igw", &ec2.InternetGatewayArgs{
		VpcId: vpc.ID(),
	})
	if err != nil {
		return err
	}
	routeTable, err := ec2.NewRouteTable(ctx, "minecraft-rt", &ec2.RouteTableArgs{
		VpcId: vpc.ID(),
		Routes: ec2.RouteTableRouteArray{
			ec2.RouteTableRouteArgs{
				CidrBlock: pulumi.String("0.0.0.0/0"),
				GatewayId: internetGateway.ID(),
			},
		},
	})
	if err != nil {
		return err
	}
	_, err = ec2.NewRouteTableAssociation(ctx, "minecraft-rt-association", &ec2.RouteTableAssociationArgs{
		SubnetId:     subnet.ID(),
		RouteTableId: routeTable.ID(),
	})
	if err != nil {
		return err
	}
	securityGroup, err := ec2.NewSecurityGroup(ctx, "minecraft-sg", &ec2.SecurityGroupArgs{
		VpcId: vpc.ID(),
		Ingress: ec2.SecurityGroupIngressArray{
			ec2.SecurityGroupIngressArgs{
				Protocol:   pulumi.String("tcp"),
				FromPort:   pulumi.Int(25565),
				ToPort:     pulumi.Int(25565),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
			ec2.SecurityGroupIngressArgs{
				Protocol:   pulumi.String("tcp"),
				FromPort:   pulumi.Int(22),
				ToPort:     pulumi.Int(22),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
		Egress: ec2.SecurityGroupEgressArray{
			ec2.SecurityGroupEgressArgs{
				Protocol:   pulumi.String("-1"),
				FromPort:   pulumi.Int(0),
				ToPort:     pulumi.Int(0),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
	})
	if err != nil {
		return err
	}
	ami, err := ec2.LookupAmi(ctx, &ec2.LookupAmiArgs{
		Filters: []ec2.GetAmiFilter{
			{
				Name:   "name",
				Values: []string{"ubuntu-minimal/images/hvm-ssd/ubuntu-jammy-22.04*"},
			},
			{
				Name:   "architecture",
				Values: []string{"x86_64"},
			},
		},
		MostRecent: pulumi.BoolRef(true),
		Owners:     []string{"099720109477"},
	}, nil)
	if err != nil {
		return err
	}
	pubKey, err := os.ReadFile("../minecraft.pub")
	if err != nil {
		return err
	}

	keypair, err := ec2.NewKeyPair(ctx, "minecraft-keypair", &ec2.KeyPairArgs{
		KeyName:   pulumi.String("minecraft-keypair"),
		PublicKey: pulumi.String(pubKey),
	})
	if err != nil {
		return err
	}
	userdata, err := os.ReadFile("../cloud-init.yaml")
	if err != nil {
		return err
	}

	instance, err := ec2.NewInstance(ctx, "minecraft-instance", &ec2.InstanceArgs{
		InstanceType:   pulumi.String("t3.xlarge"),
		Ami:            pulumi.String(ami.Id),
		SubnetId:       subnet.ID(),
		SecurityGroups: pulumi.StringArray{securityGroup.ID()},
		KeyName:        keypair.KeyName,
		UserData:       pulumi.String(userdata),
	})
	if err != nil {
		return err
	}
	ctx.Export("minecraft_vm_ip", instance.PublicIp)
	return nil
}

func createOrSelectMinecraftStack(ctx context.Context, stackName string, projectName string) (auto.Stack, error) {
	s, err := auto.UpsertStackInlineSource(ctx, stackName, projectName, createMincraftServer)
	if err != nil {
		ErrorLogger.Fatalf("Failed to create stack: %v", err)
	}
	InfoLogger.Println("Created/Selected stack: ", stackName)

	w := s.Workspace()

	DebugLogger.Println("Installing the AWS plugin")

	// for inline source programs, we must manage plugins ourselves
	err = w.InstallPlugin(ctx, "aws", "v5.28.0")
	if err != nil {
		ErrorLogger.Fatalf("Failed to install program plugins: %v\n", err)
	}

	DebugLogger.Println("Successfully installed AWS plugin")

	// set stack configuration specifying the AWS region to deploy
	s.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: "eu-central-1"})

	InfoLogger.Println("Successfully set config")
	InfoLogger.Println("Starting refresh")

	_, err = s.Refresh(ctx)
	if err != nil {
		ErrorLogger.Fatalf("Failed to refresh stack: %v\n", err)
	}

	InfoLogger.Println("Refresh succeeded!")
	return s, nil
}
func getDeploymentStatus(organization, project, stack, deploymentID, accessToken, nextToken string) (string, error) {
	baseURL := "https://api.pulumi.com/api/preview"
	url := fmt.Sprintf("%s/%s/%s/%s/deployments/%s", baseURL, organization, project, stack, deploymentID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", accessToken))

	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}
	return string(body), nil
}

type DeploymentStatus struct {
	Status string `json:"status"`
}

func main() {
	var stackName string

	var createCmd = &cobra.Command{
		Use:   "create",
		Short: "Create a new minecraft server",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				ErrorLogger.Println("Please provide a name for project")
				return
			}
			InfoLogger.Println("Creating item with name:", args[0])
			projectName := args[0]
			ctx := context.Background()
			s, err := createOrSelectMinecraftStack(ctx, stackName, projectName)
			if err != nil {
				ErrorLogger.Fatalf("Failed to create stack: %v", err)
			}

			InfoLogger.Println("Starting update")

			stdoutStreamer := optup.ProgressStreams(os.Stdout)

			res, err := s.Up(ctx, stdoutStreamer)
			if err != nil {
				ErrorLogger.Fatalf("Failed to update stack: %v\n\n", err)
			}

			InfoLogger.Println("Update succeeded!")

			ip, ok := res.Outputs["minecraft_vm_ip"].Value.(string)
			if !ok {
				ErrorLogger.Fatalf("Failed to unmarshall IP address")
			}

			InfoLogger.Printf("IP: %s\n", ip)

		},
	}

	createCmd.Flags().StringVarP(&stackName, "stack", "s", "dev", "The name of the stack")

	var destroyCommand = &cobra.Command{
		Use:   "destroy",
		Short: "Destroy a minecraft server",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				ErrorLogger.Println("Please provide a name for project")
				return
			}
			InfoLogger.Println("Destroying item with name:", args[0])
			projectName := args[0]
			ctx := context.Background()
			s, err := createOrSelectMinecraftStack(ctx, stackName, projectName)
			if err != nil {
				ErrorLogger.Fatalf("Failed to create stack: %v", err)
			}
			InfoLogger.Println("Starting destroy")
			stdoutStreamer := optdestroy.ProgressStreams(os.Stdout)
			_, err = s.Destroy(ctx, stdoutStreamer)
			if err != nil {
				ErrorLogger.Fatalf("Failed to destroy stack: %v\n\n", err)
			}
			InfoLogger.Println("Destroy succeeded!")
		},
	}
	destroyCommand.Flags().StringVarP(&stackName, "stack", "s", "dev", "The name of the stack)")

	var remoteCmd = &cobra.Command{
		Use:   "remote",
		Short: "Deploy a minecraft server using Pulumi Deployment",
		Run: func(cmd *cobra.Command, args []string) {
			pulumiAccessToken := os.Getenv("PULUMI_ACCESS_TOKEN")
			org := "dirien"
			project := "pulumi-python-minecraft"
			stack := "dev"

			id, err := startDeployment(org, project, stack, pulumiAccessToken)
			if err {
				ErrorLogger.Fatalf("Failed to start deployment")
			}
			nextToken := ""
			for {
				status, err := getDeploymentStatus(org, project, stack, id, pulumiAccessToken, nextToken)
				if err != nil {
					ErrorLogger.Fatalf("Failed to get status: %v", err)
				}

				var deploymentStatus DeploymentStatus
				err = json.Unmarshal([]byte(status), &deploymentStatus)
				if err != nil {
					ErrorLogger.Fatalf("Failed to unmarshall status: %v", err)
				}

				newStatus := deploymentStatus.Status
				if newStatus == "succeeded" {
					InfoLogger.Println("No more status available.")
					break
				}
				nextToken = newStatus
			}
			InfoLogger.Println("Deployment succeeded!")
		},
	}

	var rootCmd = &cobra.Command{Use: "server"}
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(destroyCommand)
	rootCmd.AddCommand(remoteCmd)
	rootCmd.Execute()
}

func startDeployment(organization, project, stack, accessToken string) (string, bool) {
	url := fmt.Sprintf("https://api.pulumi.com/api/preview/%s/%s/%s/deployments", organization, project, stack)

	data := map[string]interface{}{
		"operation":       "update",
		"inheritSettings": true,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		ErrorLogger.Fatalf("Failed to marshal data: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		ErrorLogger.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("token %s", accessToken))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		ErrorLogger.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		ErrorLogger.Fatalf("Failed to read response: %v", err)
	}

	var jsonResponse map[string]interface{}
	err = json.Unmarshal([]byte(string(body)), &jsonResponse)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return "", true
	}

	id, ok := jsonResponse["id"].(string)
	if !ok {
		ErrorLogger.Fatalf("Failed to unmarshall ID")
	}
	return id, false
}
