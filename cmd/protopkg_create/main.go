package main

import (
	"context"
	"crypto/x509"
	"flag"
	"fmt"
	"os"

	pppb "github.com/stackb/apis/build/stack/protobuf/package/v1alpha1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type flagName string

const (
	protoPackageFileFlagName      flagName = "proto_package_file"
	packagesServerAddressFlagName flagName = "packages_server_address"
	protoOutputFileFlagName       flagName = "proto_out"
	jsonOutputFileFlagName        flagName = "json_out"
)

var (
	protoPackageFile      = flag.String(string(protoPackageFileFlagName), "", "path to the proto package file")
	packagesServerAddress = flag.String(string(packagesServerAddressFlagName), "", "address of the packages server")
	protoOutputFile       = flag.String(string(protoOutputFileFlagName), "", "path of file to write the generated proto file")
	jsonOutputFile        = flag.String(string(jsonOutputFileFlagName), "", "path of file to write the generated json file")
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
	}
}

func run() error {
	flag.Parse()

	pkg, err := readProtoPackageFile(protoPackageFileFlagName, *protoPackageFile)
	if err != nil {
		return err
	}

	client, conn, err := createPackagesClient(*packagesServerAddress)
	if err != nil {
		return err
	}
	defer conn.Close()

	response, err := sendProtoPackage(pkg, client)
	if err != nil {
		return err
	}

	if protoOutputFile != nil {
		if err := writeProtoOutputFile(response, *protoOutputFile); err != nil {
			return err
		}
	}
	if jsonOutputFile != nil {
		if err := writeJsonOutputFile(response, *jsonOutputFile); err != nil {
			return err
		}
	}

	return nil
}

func createPackagesClient(address string) (pppb.PackagesClient, *grpc.ClientConn, error) {
	pool, err := x509.SystemCertPool()
	if err != nil {
		return nil, nil, fmt.Errorf("getting system x509 cert pool: %w", err)
	}

	// var options []grpc.ClientConn
	creds := credentials.NewClientTLSFromCert(pool, "")
	conn, err := grpc.Dial(address,
		grpc.WithTransportCredentials(creds),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("dialing connection: %w", err)
	}

	return pppb.NewPackagesClient(conn), conn, nil
}

func sendProtoPackage(pkg *pppb.ProtoPackage, client pppb.PackagesClient) (proto.Message, error) {
	requests := []*pppb.CreateProtoPackageRequest{
		{Pkg: pkg},
	}
	ctx := context.Background()
	stream, err := client.CreateProtoPackage(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating client stream call: %w", err)
	}
	for _, createRequest := range requests {
		if err := stream.Send(createRequest); err != nil {
			return nil, fmt.Errorf("sending package %s: %w", createRequest.Pkg.Name, err)
		}
	}
	operation, err := stream.CloseAndRecv()
	if err != nil {
		return nil, fmt.Errorf("close-recv stream call: %w", err)
	}
	return operation, nil
}

func readProtoPackageFile(flag flagName, filename string) (*pppb.ProtoPackage, error) {
	if filename == "" {
		return nil, fmt.Errorf("flag required but not provided: %s", flag)
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", flag, err)
	}
	var msg pppb.ProtoPackage
	if err := proto.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("unmarshaling %s: %w", flag, err)
	}
	return &msg, nil
}

func writeProtoOutputFile(msg proto.Message, filename string) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshaling generated data: %v", err)
	}
	if err := os.WriteFile(filename, data, os.ModePerm); err != nil {
		return fmt.Errorf("writing proto file: %w", err)
	}
	return nil
}

func writeJsonOutputFile(msg proto.Message, filename string) error {
	marshaler := protojson.MarshalOptions{
		Multiline: true,
		Indent:    "  ",
	}
	jsonstr, err := marshaler.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshaling json: %w", err)
	}
	if err := os.WriteFile(filename, []byte(jsonstr), os.ModePerm); err != nil {
		return fmt.Errorf("writing json file: %w", err)
	}
	return nil
}
