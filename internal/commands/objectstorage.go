package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/objectstorage"
)

const minObjectStorageGB = 60

// NewObjectStorageCmd returns the 'object-storage' cobra command.
func NewObjectStorageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "object-storage",
		Aliases: []string{"os"},
		Short:   "Manage Ceph object storage instances",
	}
	cmd.AddCommand(newOSListCmd())
	cmd.AddCommand(newOSGetCmd())
	cmd.AddCommand(newOSCreateCmd())
	cmd.AddCommand(newOSDeleteCmd())
	cmd.AddCommand(newOSResizeCmd())
	cmd.AddCommand(newOSCredentialsCmd())
	cmd.AddCommand(newOSBucketCmd())
	cmd.AddCommand(newOSObjectCmd())
	return cmd
}

func newOSListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List object storage instances",
		Example: `  zcp object-storage list
  zcp object-storage list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			stores, err := svc.List(ctx)
			if err != nil {
				return fmt.Errorf("object-storage list: %w", err)
			}

			headers := []string{"SLUG", "NAME", "SIZE (GB)", "USED (GB)", "STATUS", "REGION", "CREATED"}
			rows := make([][]string, 0, len(stores))
			for _, s := range stores {
				regionName := ""
				if s.Region != nil {
					regionName = s.Region.Name
				}
				rows = append(rows, []string{
					s.Slug,
					s.Name,
					s.Size.String(),
					s.UsedSpace.String(),
					s.Status,
					regionName,
					s.CreatedAt,
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
}

func newOSGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "get <slug>",
		Short:   "Get details of an object storage instance",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp object-storage get my-storage-1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			store, err := svc.Get(ctx, args[0])
			if err != nil {
				return fmt.Errorf("object-storage get: %w", err)
			}

			regionName := ""
			if store.Region != nil {
				regionName = store.Region.Name
			}
			projectName := ""
			if store.Project != nil {
				projectName = store.Project.Name
			}

			headers := []string{"FIELD", "VALUE"}
			rows := [][]string{
				{"Slug", store.Slug},
				{"Name", store.Name},
				{"Status", store.Status},
				{"Size (GB)", store.Size.String()},
				{"Used (GB)", store.UsedSpace.String()},
				{"S3 Endpoint", store.S3Endpoint()},
				{"Access Key", store.APIKey},
				{"Secret Key", store.APISecret},
				{"Region", regionName},
				{"Project", projectName},
				{"Created", store.CreatedAt},
			}
			return printer.PrintTable(headers, rows)
		},
	}
}

func newOSCreateCmd() *cobra.Command {
	var name, project, cloudProvider, region, billingCycle, storageCategory, plan, coupon string
	var storageGB int

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new object storage instance",
		Example: `  zcp object-storage create --name my-storage --region yul-1 --billing-cycle hourly --storage-gb 100
  zcp object-storage create --name my-storage --region yul-1 --billing-cycle hourly --plan my-plan
  zcp object-storage create --name my-storage --region yul-1 --billing-cycle hourly --storage-gb 100 --project my-project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			project = resolveProject(project)
			if project == "" {
				return fmt.Errorf("--project is required (or set ZCP_PROJECT)")
			}
			cloudProvider = resolveCloudProvider(cloudProvider)
			if cloudProvider == "" {
				cloudProvider = "ceph"
			}
			region = resolveRegion(region)
			if region == "" {
				return fmt.Errorf("--region is required (or set ZCP_REGION)")
			}
			if billingCycle == "" {
				return fmt.Errorf("--billing-cycle is required")
			}
			if storageGB < 0 {
				return fmt.Errorf("--storage-gb cannot be negative")
			}
			if plan == "" && storageGB == 0 {
				return fmt.Errorf("either --plan or --storage-gb is required")
			}
			if plan != "" && storageGB != 0 {
				return fmt.Errorf("--plan and --storage-gb are mutually exclusive")
			}
			if storageGB > 0 && storageGB < minObjectStorageGB {
				return fmt.Errorf("--storage-gb must be at least %d", minObjectStorageGB)
			}

			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			req := objectstorage.CreateRequest{
				Name:            name,
				Project:         project,
				CloudProvider:   cloudProvider,
				Region:          region,
				BillingCycle:    billingCycle,
				StorageCategory: storageCategory,
				Plan:            plan,
				Coupon:          coupon,
			}
			if storageGB > 0 {
				req.CustomPlan = &objectstorage.CustomPlan{Storage: storageGB}
			}

			store, err := svc.Create(ctx, req)
			if err != nil {
				return fmt.Errorf("object-storage create: %w", err)
			}

			regionName := ""
			if store.Region != nil {
				regionName = store.Region.Name
			}
			headers := []string{"SLUG", "NAME", "SIZE (GB)", "STATUS", "REGION", "CREATED"}
			rows := [][]string{{
				store.Slug,
				store.Name,
				store.Size.String(),
				store.Status,
				regionName,
				store.CreatedAt,
			}}
			return printer.PrintTable(headers, rows)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Object storage name (required)")
	cmd.Flags().StringVar(&project, "project", "", "Project slug (or set ZCP_PROJECT)")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider slug (default: ceph)")
	cmd.Flags().StringVar(&region, "region", "", "Region slug, e.g. yul-1 or yow-1 (or set ZCP_REGION)")
	cmd.Flags().StringVar(&billingCycle, "billing-cycle", "", "Billing cycle slug, e.g. hourly (required)")
	cmd.Flags().StringVar(&storageCategory, "storage-category", "premium-ssd", "Storage category slug")
	cmd.Flags().StringVar(&plan, "plan", "", "Plan slug (mutually exclusive with --storage-gb)")
	cmd.Flags().IntVar(&storageGB, "storage-gb", 0, "Custom storage size in GB, minimum 60 (mutually exclusive with --plan)")
	cmd.Flags().StringVar(&coupon, "coupon", "", "Coupon code")
	return cmd
}

func newOSCredentialsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "credentials <slug>",
		Short: "Show S3 credentials for an object storage instance",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp object-storage credentials my-storage-1
  zcp object-storage credentials my-storage-1 --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			store, err := svc.Get(ctx, args[0])
			if err != nil {
				return fmt.Errorf("object-storage credentials: %w", err)
			}

			headers := []string{"FIELD", "VALUE"}
			rows := [][]string{
				{"S3 Endpoint", store.S3Endpoint()},
				{"Access Key", store.APIKey},
				{"Secret Key", store.APISecret},
			}
			return printer.PrintTable(headers, rows)
		},
	}
}

func newOSDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <slug>",
		Short: "Delete an object storage instance",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp object-storage delete my-storage-1
  zcp object-storage delete my-storage-1 -y`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			if !autoApproved(cmd) {
				fmt.Fprintf(os.Stderr, "Delete object storage %q? All data will be permanently lost. [y/N]: ", slug)
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
				answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
				if answer != "y" && answer != "yes" {
					fmt.Fprintln(os.Stderr, "Aborted.")
					return nil
				}
			}

			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			if err := svc.Delete(ctx, slug); err != nil {
				return fmt.Errorf("object-storage delete: %w", err)
			}
			printer.Fprintf("Object storage %q deleted.\n", slug)
			return nil
		},
	}
	return cmd
}

func newOSResizeCmd() *cobra.Command {
	var storageGB int

	cmd := &cobra.Command{
		Use:     "resize <slug>",
		Short:   "Resize the storage allocation of an object storage instance",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp object-storage resize my-storage-1 --storage-gb 200`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if storageGB <= 0 || storageGB < minObjectStorageGB {
				return fmt.Errorf("--storage-gb must be at least %d", minObjectStorageGB)
			}

			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			store, err := svc.Resize(ctx, args[0], storageGB)
			if err != nil {
				return fmt.Errorf("object-storage resize: %w", err)
			}

			headers := []string{"SLUG", "NAME", "SIZE (GB)", "STATUS"}
			rows := [][]string{{
				store.Slug,
				store.Name,
				store.Size.String(),
				store.Status,
			}}
			return printer.PrintTable(headers, rows)
		},
	}
	cmd.Flags().IntVar(&storageGB, "storage-gb", 0, "New storage size in GB, minimum 60 (required)")
	_ = cmd.MarkFlagRequired("storage-gb")
	return cmd
}

// newOSBucketCmd returns the 'object-storage bucket' subcommand group.
func newOSBucketCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bucket",
		Short: "Manage buckets within an object storage instance",
	}
	cmd.AddCommand(newOSBucketListCmd())
	cmd.AddCommand(newOSBucketGetCmd())
	cmd.AddCommand(newOSBucketCreateCmd())
	cmd.AddCommand(newOSBucketDeleteCmd())
	cmd.AddCommand(newOSBucketSetACLCmd())
	return cmd
}

// newOSObjectCmd returns the 'object-storage object' subcommand group.
func newOSObjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "object",
		Short: "Manage objects in a bucket",
	}
	cmd.AddCommand(newOSObjectListCmd())
	cmd.AddCommand(newOSObjectGetCmd())
	cmd.AddCommand(newOSObjectPutCmd())
	cmd.AddCommand(newOSObjectDeleteCmd())
	return cmd
}

func newOSBucketListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <storage-slug>",
		Short: "List buckets in an object storage instance",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp object-storage bucket list my-storage-1
  zcp object-storage bucket list my-storage-1 --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			buckets, err := svc.ListBuckets(ctx, args[0])
			if err != nil {
				return fmt.Errorf("object-storage bucket list: %w", err)
			}

			headers := []string{"SLUG", "NAME", "OBJECTS", "SIZE (GB)", "STATUS", "CREATED"}
			rows := make([][]string, 0, len(buckets))
			for _, b := range buckets {
				rows = append(rows, []string{
					b.Slug,
					b.Name,
					fmt.Sprintf("%d", b.ObjectCount),
					b.Size.String(),
					b.Status,
					b.CreatedAt,
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
}

func newOSBucketGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "get <storage-slug> <bucket-slug>",
		Short:   "Get details of a bucket",
		Args:    cobra.ExactArgs(2),
		Example: `  zcp object-storage bucket get my-storage-1 my-bucket`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			bucket, err := svc.GetBucket(ctx, args[0], args[1])
			if err != nil {
				return fmt.Errorf("object-storage bucket get: %w", err)
			}

			headers := []string{"FIELD", "VALUE"}
			rows := [][]string{
				{"Slug", bucket.Slug},
				{"Name", bucket.Name},
				{"Status", bucket.Status},
				{"Objects", fmt.Sprintf("%d", bucket.ObjectCount)},
				{"Size (GB)", bucket.Size.String()},
				{"Created", bucket.CreatedAt},
			}
			return printer.PrintTable(headers, rows)
		},
	}
}

func newOSBucketCreateCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:     "create <storage-slug>",
		Short:   "Create a new bucket in an object storage instance",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp object-storage bucket create my-storage-1 --name my-bucket`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}

			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			bucket, err := svc.CreateBucket(ctx, args[0], name)
			if err != nil {
				return fmt.Errorf("object-storage bucket create: %w", err)
			}

			headers := []string{"SLUG", "NAME", "STATUS", "CREATED"}
			rows := [][]string{{
				bucket.Slug,
				bucket.Name,
				bucket.Status,
				bucket.CreatedAt,
			}}
			return printer.PrintTable(headers, rows)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Bucket name (required)")
	return cmd
}

func newOSBucketDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <storage-slug> <bucket-slug>",
		Short: "Delete a bucket from an object storage instance",
		Args:  cobra.ExactArgs(2),
		Example: `  zcp object-storage bucket delete my-storage-1 my-bucket
  zcp object-storage bucket delete my-storage-1 my-bucket -y`,
		RunE: func(cmd *cobra.Command, args []string) error {
			storageSlug, bucketSlug := args[0], args[1]
			if !autoApproved(cmd) {
				fmt.Fprintf(os.Stderr, "Delete bucket %q from %q? All objects will be permanently lost. [y/N]: ", bucketSlug, storageSlug)
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
				answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
				if answer != "y" && answer != "yes" {
					fmt.Fprintln(os.Stderr, "Aborted.")
					return nil
				}
			}

			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			if err := svc.DeleteBucket(ctx, storageSlug, bucketSlug); err != nil {
				return fmt.Errorf("object-storage bucket delete: %w", err)
			}
			printer.Fprintf("Bucket %q deleted from %q.\n", bucketSlug, storageSlug)
			return nil
		},
	}
	return cmd
}

func newOSBucketSetACLCmd() *cobra.Command {
	var acl string

	cmd := &cobra.Command{
		Use:   "set-acl <storage-slug> <bucket-slug>",
		Short: "Set the access control on a bucket",
		Args:  cobra.ExactArgs(2),
		Example: `  zcp object-storage bucket set-acl my-storage-1 my-bucket --acl public-read
  zcp object-storage bucket set-acl my-storage-1 my-bucket --acl private`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if acl == "" {
				return fmt.Errorf("--acl is required (values: private, public-read, public-read-write, authenticated-read)")
			}

			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			bucket, err := svc.SetBucketACL(ctx, args[0], args[1], acl)
			if err != nil {
				return fmt.Errorf("object-storage bucket set-acl: %w", err)
			}

			headers := []string{"SLUG", "NAME", "STATUS"}
			rows := [][]string{{bucket.Slug, bucket.Name, bucket.Status}}
			return printer.PrintTable(headers, rows)
		},
	}
	cmd.Flags().StringVar(&acl, "acl", "", "ACL value: private, public-read, public-read-write, authenticated-read (required)")
	return cmd
}

func newOSObjectListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <storage-slug> <bucket-slug>",
		Short: "List objects in a bucket",
		Args:  cobra.ExactArgs(2),
		Example: `  zcp object-storage object list my-storage-1 my-bucket
  zcp object-storage object list my-storage-1 my-bucket --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			objects, err := svc.ListObjects(ctx, args[0], args[1])
			if err != nil {
				return fmt.Errorf("object-storage object list: %w", err)
			}

			headers := []string{"KEY", "SIZE", "PERMISSION", "LAST MODIFIED"}
			rows := make([][]string, 0, len(objects))
			for _, o := range objects {
				rows = append(rows, []string{
					o.Key,
					o.Size,
					o.Permission,
					o.LastModified,
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
}

func newOSObjectGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "get <storage-slug> <bucket-slug> <object-key>",
		Short:   "Get details of an object in a bucket",
		Args:    cobra.ExactArgs(3),
		Example: `  zcp object-storage object get my-storage-1 my-bucket my-file.txt`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			obj, err := svc.GetObject(ctx, args[0], args[1], args[2])
			if err != nil {
				return fmt.Errorf("object-storage object get: %w", err)
			}

			headers := []string{"FIELD", "VALUE"}
			rows := [][]string{
				{"Key", obj.Key},
				{"Name", obj.Name},
				{"Size", obj.Size},
				{"Permission", obj.Permission},
				{"Last Modified", obj.LastModified},
			}
			return printer.PrintTable(headers, rows)
		},
	}
}

func newOSObjectPutCmd() *cobra.Command {
	var (
		key         string
		contentType string
	)

	cmd := &cobra.Command{
		Use:   "put <storage-slug> <bucket-slug> <local-file>",
		Short: "Upload a local file to a bucket",
		Args:  cobra.ExactArgs(3),
		Example: `  zcp object-storage object put my-storage-1 my-bucket ./report.pdf
  zcp object-storage object put my-storage-1 my-bucket ./logo.png --key images/logo.png
  zcp object-storage object put my-storage-1 my-bucket ./data.bin --content-type application/octet-stream`,
		RunE: func(cmd *cobra.Command, args []string) error {
			storageSlug, bucketSlug, localFile := args[0], args[1], args[2]

			_, client, _, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			size, err := svc.PutObject(ctx, storageSlug, bucketSlug, localFile, key, contentType)
			if err != nil {
				return fmt.Errorf("object-storage object put: %w", err)
			}

			effectiveKey := key
			if effectiveKey == "" {
				effectiveKey = filepath.Base(localFile)
			}
			fmt.Fprintf(os.Stdout, "Uploaded %q to %s/%s (%d bytes)\n", localFile, bucketSlug, effectiveKey, size)
			return nil
		},
	}
	cmd.Flags().StringVar(&key, "key", "", "Remote object key (defaults to the local filename)")
	cmd.Flags().StringVar(&contentType, "content-type", "", "Content-Type header (auto-detected from extension when omitted)")
	return cmd
}

func newOSObjectDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <storage-slug> <bucket-slug> <object-key>",
		Short: "Delete an object from a bucket",
		Args:  cobra.ExactArgs(3),
		Example: `  zcp object-storage object delete my-storage-1 my-bucket report.pdf
  zcp object-storage object delete my-storage-1 my-bucket images/logo.png -y`,
		RunE: func(cmd *cobra.Command, args []string) error {
			storageSlug, bucketSlug, objectKey := args[0], args[1], args[2]

			if !autoApproved(cmd) {
				fmt.Fprintf(os.Stderr, "Delete object %q from bucket %q? [y/N]: ", objectKey, bucketSlug)
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
				answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
				if answer != "y" && answer != "yes" {
					fmt.Fprintln(os.Stderr, "Aborted.")
					return nil
				}
			}

			_, client, _, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			if err := svc.DeleteObject(ctx, storageSlug, bucketSlug, objectKey); err != nil {
				return fmt.Errorf("object-storage object delete: %w", err)
			}

			fmt.Fprintf(os.Stdout, "Deleted %q from bucket %q.\n", objectKey, bucketSlug)
			return nil
		},
	}
}
