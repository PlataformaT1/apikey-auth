package main

import (
	"log"
	"os"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type DeployStackProps struct {
	awscdk.StackProps
}

func NewDeployStack(scope constructs.Construct, id string, props *DeployStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	logChan := getEnvOrDefault("USER_VAR_LOG_CHAN", "default-log-channel")
	logLevel := getEnvOrDefault("USER_VAR_LOG_LEVEL", "info")

	mongoURI := os.Getenv("USER_VAR_DB_MONGO_URI")
	if mongoURI == "" {
		log.Fatalf("La variable de entorno USER_VAR_DB_MONGO_URI no está definida")
	}

	envs := &map[string]*string{
		/*"USER_VAR_LOG_CHAN":     jsii.String("Stdout"),
		"USER_VAR_LOG_LEVEL":    jsii.String("INFO"),
		"USER_VAR_DB_MONGO_URI": jsii.String("mongodb://t1pnonpci:qK!X5oNqP0b2@t1pnonpcidocumentdbcluster-f6py01dyecc7.cluster-ckh5zkjba5nq.us-east-1.docdb.amazonaws.com:27017/?replicaSet=rs0&readPreference=secondaryPreferred&retryWrites=false"),*/
		"USER_VAR_LOG_CHAN":     jsii.String(logChan),
		"USER_VAR_LOG_LEVEL":    jsii.String(logLevel),
		"USER_VAR_DB_MONGO_URI": jsii.String(mongoURI),
	}

	// Lookup VPC
	vpcID := getEnvOrDefault("USER_VAR_VPC_ID", "")
	vpc := awsec2.Vpc_FromLookup(stack, jsii.String("VpcMongo"), &awsec2.VpcLookupOptions{
		//VpcId: jsii.String("vpc-06d959fa883f7099f"),
		VpcId: jsii.String(vpcID),
	})

	// Lookup Subnet
	subnetIDs := getEnvOrDefault("USER_VAR_SUBNET_IDS", "")
	//subnet := awsec2.Subnet_FromSubnetId(stack, jsii.String("TargetSubnet"), jsii.String("subnet-048e1b539693ca677"))
	subnet := awsec2.Subnet_FromSubnetId(stack, jsii.String("TargetSubnet"), jsii.String(subnetIDs))

	// Define subnet selection
	subnetSelection := &awsec2.SubnetSelection{
		Subnets: &[]awsec2.ISubnet{subnet},
	}

	// Lookup Security Group
	securityGroupIDs := getEnvOrDefault("USER_VAR_SECURITY_GROUP_IDS", "")
	//securityGroup := awsec2.SecurityGroup_FromSecurityGroupId(stack, jsii.String("TargetSG"), jsii.String("sg-069a39550d6ce94b2"), nil)
	securityGroup := awsec2.SecurityGroup_FromSecurityGroupId(stack, jsii.String("TargetSG"), jsii.String(securityGroupIDs), nil)

	// Create Lambda function with VPC and specific subnet
	awslambda.NewFunction(stack, jsii.String("AuthorizerApiKeyFunction"), &awslambda.FunctionProps{
		FunctionName:      jsii.String("authorizerApiKeyFunction"),
		Runtime:           awslambda.Runtime_PROVIDED_AL2023(),
		Architecture:      awslambda.Architecture_X86_64(),
		Code:              awslambda.AssetCode_FromAsset(jsii.String("./authorizer/app/cmd"), nil),
		Handler:           jsii.String("bootstrap"),
		MemorySize:        jsii.Number(128),
		Timeout:           awscdk.Duration_Seconds(jsii.Number(60)),
		Vpc:               vpc,
		SecurityGroups:    &[]awsec2.ISecurityGroup{securityGroup},
		AllowPublicSubnet: jsii.Bool(true),
		VpcSubnets:        subnetSelection,
		Environment:       envs,
	})

	return stack
}

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)

	NewDeployStack(app, "AuthorizerApiKey", &DeployStackProps{
		awscdk.StackProps{
			Env: env(),
		},
	})

	app.Synth(nil)
}

func env() *awscdk.Environment {
	// Para cuenta AWS (línea 79 aprox.)
	awsAccount := getEnvOrDefault("USER_VAR_AWS_ACCOUNT_ID", "")
	if awsAccount == "" {
		log.Fatalf("La variable de entorno USER_VAR_AWS_ACCOUNT_ID no está definida")
	}

	// Para región AWS (línea 80 aprox.)
	awsRegion := getEnvOrDefault("USER_VAR_AWS_REGION", "us-east-1")

	return &awscdk.Environment{
		/*Account: jsii.String("888577062296"),
		Region:  jsii.String("us-east-1"),*/
		Account: jsii.String(awsAccount),
		Region:  jsii.String(awsRegion),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
