package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/zsoftly/zcp-cli/internal/output"
	"github.com/zsoftly/zcp-cli/pkg/api/apierrors"
	"github.com/zsoftly/zcp-cli/pkg/api/objectstorage"
	"github.com/zsoftly/zcp-cli/pkg/api/plan"
	"github.com/zsoftly/zcp-cli/pkg/api/storagecategory"
	"github.com/zsoftly/zcp-cli/pkg/httpclient"
)

const minObjectStorageGB = 60

// objectStoragePlanCategory looks up an Object Storage plan by slug and returns
// the storage-category slug it is configured for. The create API requires a
// storage_category that matches the plan, so this lets the CLI fill it in
// automatically instead of making the user know the pairing.
func objectStoragePlanCategory(ctx context.Context, client *httpclient.Client, planSlug, region string) (string, error) {
	plans, err := plan.NewService(client).List(ctx, plan.ServiceObjectStorage, region)
	if err != nil {
		return "", fmt.Errorf("looking up plan %q: %w", planSlug, err)
	}
	var catID string
	for _, p := range plans {
		if p.Slug == planSlug {
			catID = p.StorageCategoryID
			break
		}
	}
	if catID == "" {
		return "", fmt.Errorf("plan %q not found among Object Storage plans (see 'zcp plan object-storage')", planSlug)
	}
	cats, err := storagecategory.NewService(client).List(ctx, region)
	if err != nil {
		return "", fmt.Errorf("resolving storage category for plan %q: %w", planSlug, err)
	}
	for _, c := range cats {
		if c.ID == catID {
			return c.Slug, nil
		}
	}
	return "", fmt.Errorf("storage category for plan %q could not be resolved", planSlug)
}

// NewObjectStorageCmd returns the 'object-storage' cobra command.
func NewObjectStorageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "object-storage",
		Aliases: []string{"os"},
		Short:   "Manage object storage instances",
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

			// Object-storage regions (os-yul/os-yow) differ from the profile's
			// compute region default, so resolve region from --region/ZCP_REGION
			// only — applying the compute default (e.g. yul-1) would filter to a
			// region that holds no object storage and silently return nothing.
			// Project is region-agnostic, so its profile default is fine.
			flagRegion, _ := cmd.Flags().GetString("region")
			region := strings.TrimSpace(resolveRegion(flagRegion))
			if region == "" {
				return fmt.Errorf("--region is required (or set ZCP_REGION)")
			}
			_, project := scopedRegionProject(cmd)
			stores, err := svc.List(ctx, region, project)
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
		Args:    exactArgs(1),
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
		Example: `  zcp object-storage create --name my-storage --region os-yul --billing-cycle hourly --storage-gb 100
  zcp object-storage create --name my-storage --region os-yow --billing-cycle hourly --plan my-plan
  zcp object-storage create --name my-storage --region os-yul --billing-cycle hourly --storage-gb 100 --project default-9`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			project = resolveProject(project)
			if project == "" {
				return fmt.Errorf("--project is required (or set ZCP_PROJECT)")
			}
			cloudProvider = cloudProviderFlagOrEnv(cloudProvider)
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

			// When a plan is given without an explicit --storage-category, derive
			// the category the plan requires (the API rejects a mismatch and
			// requires a non-empty category even with a plan).
			if plan != "" && !cmd.Flags().Changed("storage-category") {
				derived, derr := objectStoragePlanCategory(ctx, client, plan, region)
				if derr != nil {
					return fmt.Errorf("object-storage create: %w", derr)
				}
				storageCategory = derived
			}

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
	cmd.Flags().StringVar(&region, "region", "", "Object-storage region slug, e.g. os-yul or os-yow (or set ZCP_REGION)")
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
		Args:  exactArgs(1),
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
		Args:  exactArgs(1),
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
				if apierrors.IsResourceNotFound(err) {
					fmt.Fprintf(os.Stderr, "Object storage %q not found — already deleted.\n", slug)
					return nil
				}
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
		Args:    exactArgs(1),
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
	cmd.AddCommand(newOSBucketVersioningCmd())
	cmd.AddCommand(newOSBucketPolicyCmd())
	cmd.AddCommand(newOSBucketTagCmd())
	cmd.AddCommand(newOSBucketEncryptionCmd())
	cmd.AddCommand(newOSBucketLifecycleCmd())
	cmd.AddCommand(newOSBucketCORSCmd())
	cmd.AddCommand(newOSBucketUploadsCmd())
	cmd.AddCommand(newOSBucketEmptyCmd())
	return cmd
}

func newOSBucketUploadsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uploads",
		Short: "Manage incomplete multipart uploads (storage consumed by failed large uploads)",
	}
	var prefix string
	listCmd := &cobra.Command{
		Use:   "list <storage-slug> <bucket-slug>",
		Short: "List incomplete multipart uploads in a bucket",
		Args:  exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			ups, err := svc.ListIncompleteUploads(ctx, args[0], args[1], prefix)
			if err != nil {
				return fmt.Errorf("object-storage bucket uploads list: %w", err)
			}
			rows := make([][]string, 0, len(ups))
			for _, u := range ups {
				rows = append(rows, []string{u.Key, u.UploadID, fmt.Sprintf("%d", u.Size), u.Initiated.Format(time.RFC3339)})
			}
			return printer.PrintTable([]string{"KEY", "UPLOAD ID", "SIZE", "INITIATED"}, rows)
		},
	}
	listCmd.Flags().StringVar(&prefix, "prefix", "", "Only list uploads under this key prefix")
	cmd.AddCommand(listCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "abort <storage-slug> <bucket-slug> <object-key>",
		Short: "Abort the incomplete multipart upload(s) for a key, reclaiming storage",
		Args:  exactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, _, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			if err := svc.AbortIncompleteUpload(ctx, args[0], args[1], args[2]); err != nil {
				return fmt.Errorf("object-storage bucket uploads abort: %w", err)
			}
			fmt.Fprintf(os.Stdout, "Aborted incomplete upload(s) for %q in bucket %q.\n", args[2], args[1])
			return nil
		},
	})
	return cmd
}

func newOSBucketCORSCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "cors", Short: "Manage cross-origin (CORS) rules on a bucket"}
	cmd.AddCommand(&cobra.Command{
		Use:   "get <storage-slug> <bucket-slug>",
		Short: "Show a bucket's CORS rules (JSON; -o yaml for YAML)",
		Args:  exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			rules, err := svc.GetBucketCORS(ctx, args[0], args[1])
			if err != nil {
				return fmt.Errorf("object-storage bucket cors get: %w", err)
			}
			if rules == "" {
				fmt.Fprintln(os.Stderr, "(no CORS configuration)")
				return nil
			}
			return printRawDocument(printer, rules)
		},
	})

	var origins, methods, headers []string
	var maxAge int
	setCmd := &cobra.Command{
		Use:   "set <storage-slug> <bucket-slug>",
		Short: "Set a CORS rule (replaces existing CORS config)",
		Args:  exactArgs(2),
		Example: `  zcp object-storage bucket cors set my-store my-bucket --origin '*' --method GET --method PUT
  zcp object-storage bucket cors set my-store my-bucket --origin https://app.example.com --method GET --header '*' --max-age 3600`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(origins) == 0 || len(methods) == 0 {
				return fmt.Errorf("at least one --origin and one --method are required")
			}
			_, client, _, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			if err := svc.SetBucketCORS(ctx, args[0], args[1], origins, methods, headers, maxAge); err != nil {
				return fmt.Errorf("object-storage bucket cors set: %w", err)
			}
			fmt.Fprintf(os.Stdout, "CORS rule set on bucket %q.\n", args[1])
			return nil
		},
	}
	setCmd.Flags().StringArrayVar(&origins, "origin", nil, "Allowed origin, e.g. '*' or https://app.example.com (repeatable, required)")
	setCmd.Flags().StringArrayVar(&methods, "method", nil, "Allowed method, e.g. GET, PUT (repeatable, required)")
	setCmd.Flags().StringArrayVar(&headers, "header", nil, "Allowed request header (repeatable)")
	setCmd.Flags().IntVar(&maxAge, "max-age", 0, "Max age (seconds) browsers may cache the preflight")
	cmd.AddCommand(setCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "delete <storage-slug> <bucket-slug>",
		Short: "Remove a bucket's CORS configuration",
		Args:  exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, _, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			if err := svc.DeleteBucketCORS(ctx, args[0], args[1]); err != nil {
				return fmt.Errorf("object-storage bucket cors delete: %w", err)
			}
			fmt.Fprintf(os.Stdout, "CORS configuration removed from bucket %q.\n", args[1])
			return nil
		},
	})
	return cmd
}

// printRawDocument writes a JSON document (S3 policy/lifecycle/CORS config) to
// stdout, honoring the printer's resolved --output: JSON verbatim by default,
// converted to YAML when -o yaml is requested. (These are nested documents with
// no meaningful flat-table form, so table falls back to JSON.)
func printRawDocument(printer *output.Printer, jsonDoc string) error {
	if printer.Format() == output.FormatYAML {
		var v interface{}
		if err := json.Unmarshal([]byte(jsonDoc), &v); err == nil {
			if y, yerr := yaml.Marshal(v); yerr == nil {
				fmt.Fprint(os.Stdout, string(y))
				return nil
			}
		}
	}
	fmt.Fprintln(os.Stdout, jsonDoc)
	return nil
}

// parseKVTags converts --tag k=v flags into a map.
func parseKVTags(pairs []string) (map[string]string, error) {
	m := make(map[string]string, len(pairs))
	for _, p := range pairs {
		k, v, ok := strings.Cut(p, "=")
		if !ok || k == "" {
			return nil, fmt.Errorf("invalid --tag %q (expected key=value)", p)
		}
		m[k] = v
	}
	return m, nil
}

func newOSBucketEmptyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "empty <storage-slug> <bucket-slug>",
		Short: "Delete all objects and object versions from a bucket (keeps the bucket)",
		Args:  exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !autoApproved(cmd) {
				fmt.Fprintf(os.Stderr, "Delete ALL objects and versions in bucket %q? This cannot be undone. [y/N]: ", args[1])
				sc := bufio.NewScanner(os.Stdin)
				sc.Scan()
				if a := strings.ToLower(strings.TrimSpace(sc.Text())); a != "y" && a != "yes" {
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
			n, err := svc.EmptyBucket(ctx, args[0], args[1])
			if err != nil {
				return fmt.Errorf("object-storage bucket empty: %w", err)
			}
			fmt.Fprintf(os.Stdout, "Removed %d object/version entries from bucket %q.\n", n, args[1])
			return nil
		},
	}
	return cmd
}

func newOSBucketTagCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "tag", Short: "Manage bucket tags"}
	cmd.AddCommand(&cobra.Command{
		Use:   "get <storage-slug> <bucket-slug>",
		Short: "Show a bucket's tags",
		Args:  exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			m, err := svc.GetBucketTagging(ctx, args[0], args[1])
			if err != nil {
				return fmt.Errorf("object-storage bucket tag get: %w", err)
			}
			return printer.PrintTable([]string{"KEY", "VALUE"}, tagRows(m))
		},
	})
	var setTags []string
	setCmd := &cobra.Command{
		Use:     "set <storage-slug> <bucket-slug>",
		Short:   "Replace a bucket's tags",
		Args:    exactArgs(2),
		Example: `  zcp object-storage bucket tag set my-store my-bucket --tag env=prod --tag team=data`,
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := parseKVTags(setTags)
			if err != nil {
				return err
			}
			if len(m) == 0 {
				return fmt.Errorf("at least one --tag key=value is required")
			}
			_, client, _, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			if err := svc.SetBucketTagging(ctx, args[0], args[1], m); err != nil {
				return fmt.Errorf("object-storage bucket tag set: %w", err)
			}
			fmt.Fprintf(os.Stdout, "Set %d tag(s) on bucket %q.\n", len(m), args[1])
			return nil
		},
	}
	setCmd.Flags().StringArrayVar(&setTags, "tag", nil, "Tag as key=value (repeatable)")
	cmd.AddCommand(setCmd)
	cmd.AddCommand(&cobra.Command{
		Use:   "delete <storage-slug> <bucket-slug>",
		Short: "Remove all tags from a bucket",
		Args:  exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, _, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			if err := svc.DeleteBucketTagging(ctx, args[0], args[1]); err != nil {
				return fmt.Errorf("object-storage bucket tag delete: %w", err)
			}
			fmt.Fprintf(os.Stdout, "Tags removed from bucket %q.\n", args[1])
			return nil
		},
	})
	return cmd
}

func newOSBucketEncryptionCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "encryption", Short: "Manage default bucket encryption (SSE-S3)"}
	cmd.AddCommand(&cobra.Command{
		Use:   "status <storage-slug> <bucket-slug>",
		Short: "Show a bucket's default encryption",
		Args:  exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			alg, err := svc.GetBucketEncryption(ctx, args[0], args[1])
			if err != nil {
				return fmt.Errorf("object-storage bucket encryption status: %w", err)
			}
			if alg == "" {
				alg = "none"
			}
			return printer.PrintTable([]string{"BUCKET", "ENCRYPTION"}, [][]string{{args[1], alg}})
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "enable <storage-slug> <bucket-slug>",
		Short: "Enable default SSE-S3 encryption on a bucket",
		Args:  exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, _, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			if err := svc.SetBucketEncryption(ctx, args[0], args[1]); err != nil {
				return fmt.Errorf("object-storage bucket encryption enable: %w", err)
			}
			fmt.Fprintf(os.Stdout, "Default SSE-S3 encryption enabled on bucket %q.\n", args[1])
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "disable <storage-slug> <bucket-slug>",
		Short: "Disable default encryption on a bucket",
		Args:  exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, _, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			if err := svc.DisableBucketEncryption(ctx, args[0], args[1]); err != nil {
				return fmt.Errorf("object-storage bucket encryption disable: %w", err)
			}
			fmt.Fprintf(os.Stdout, "Default encryption disabled on bucket %q.\n", args[1])
			return nil
		},
	})
	return cmd
}

func newOSBucketLifecycleCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "lifecycle", Short: "Manage object lifecycle (expiration) rules"}
	cmd.AddCommand(&cobra.Command{
		Use:   "get <storage-slug> <bucket-slug>",
		Short: "Show a bucket's lifecycle configuration (JSON; -o yaml for YAML)",
		Args:  exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			cfg, err := svc.GetBucketLifecycle(ctx, args[0], args[1])
			if err != nil {
				return fmt.Errorf("object-storage bucket lifecycle get: %w", err)
			}
			if cfg == "" {
				fmt.Fprintln(os.Stderr, "(no lifecycle configuration)")
				return nil
			}
			return printRawDocument(printer, cfg)
		},
	})
	var days int
	var prefix string
	var noncurrentDays, abortMultipartDays int
	expireCmd := &cobra.Command{
		Use:   "expire <storage-slug> <bucket-slug>",
		Short: "Set object-expiration rules (current/noncurrent versions, incomplete uploads)",
		Args:  exactArgs(2),
		Example: `  zcp object-storage bucket lifecycle expire my-store my-bucket --days 30 --prefix logs/
  zcp object-storage bucket lifecycle expire my-store my-bucket --noncurrent-days 7 --abort-multipart-days 3`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if days <= 0 && noncurrentDays <= 0 && abortMultipartDays <= 0 {
				return fmt.Errorf("set at least one of --days, --noncurrent-days, or --abort-multipart-days (positive)")
			}
			_, client, _, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			if err := svc.SetBucketExpiry(ctx, args[0], args[1], prefix, days, noncurrentDays, abortMultipartDays); err != nil {
				return fmt.Errorf("object-storage bucket lifecycle expire: %w", err)
			}
			fmt.Fprintf(os.Stdout, "Lifecycle rule set on bucket %q.\n", args[1])
			return nil
		},
	}
	expireCmd.Flags().IntVar(&days, "days", 0, "Expire current object versions after this many days")
	expireCmd.Flags().IntVar(&noncurrentDays, "noncurrent-days", 0, "Expire noncurrent (old) versions after this many days")
	expireCmd.Flags().IntVar(&abortMultipartDays, "abort-multipart-days", 0, "Abort incomplete multipart uploads after this many days")
	expireCmd.Flags().StringVar(&prefix, "prefix", "", "Only objects under this key prefix")
	cmd.AddCommand(expireCmd)
	cmd.AddCommand(&cobra.Command{
		Use:   "delete <storage-slug> <bucket-slug>",
		Short: "Remove a bucket's lifecycle configuration",
		Args:  exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, _, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			if err := svc.DeleteBucketLifecycle(ctx, args[0], args[1]); err != nil {
				return fmt.Errorf("object-storage bucket lifecycle delete: %w", err)
			}
			fmt.Fprintf(os.Stdout, "Lifecycle configuration removed from bucket %q.\n", args[1])
			return nil
		},
	})
	return cmd
}

// tagRows turns a tag map into sorted printer rows.
func tagRows(m map[string]string) [][]string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	rows := make([][]string, 0, len(keys))
	for _, k := range keys {
		rows = append(rows, []string{k, m[k]})
	}
	return rows
}

// newOSBucketVersioningCmd groups object-versioning operations.
func newOSBucketVersioningCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "versioning",
		Short: "Manage object versioning on a bucket",
	}

	mkSet := func(use, short string, enabled bool) *cobra.Command {
		return &cobra.Command{
			Use:   use + " <storage-slug> <bucket-slug>",
			Short: short,
			Args:  exactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				_, client, _, err := buildClientAndPrinter(cmd)
				if err != nil {
					return err
				}
				svc := objectstorage.NewService(client)
				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
				defer cancel()
				if err := svc.SetBucketVersioning(ctx, args[0], args[1], enabled); err != nil {
					return fmt.Errorf("object-storage bucket versioning %s: %w", use, err)
				}
				state := "enabled"
				if !enabled {
					state = "suspended"
				}
				fmt.Fprintf(os.Stdout, "Versioning %s on bucket %q.\n", state, args[1])
				return nil
			},
		}
	}
	cmd.AddCommand(mkSet("enable", "Enable object versioning on a bucket", true))
	cmd.AddCommand(mkSet("suspend", "Suspend object versioning on a bucket", false))
	cmd.AddCommand(&cobra.Command{
		Use:   "status <storage-slug> <bucket-slug>",
		Short: "Show whether versioning is enabled on a bucket",
		Args:  exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			status, err := svc.GetBucketVersioning(ctx, args[0], args[1])
			if err != nil {
				return fmt.Errorf("object-storage bucket versioning status: %w", err)
			}
			if status == "" {
				status = "Unversioned"
			}
			return printer.PrintTable([]string{"BUCKET", "VERSIONING"}, [][]string{{args[1], status}})
		},
	})
	return cmd
}

// newOSBucketPolicyCmd groups raw S3 bucket-policy operations.
func newOSBucketPolicyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Get, set, or delete a bucket's raw S3 policy (JSON)",
		Long: `Manage the raw S3 bucket policy. For simple public/private access use
'bucket set-acl' instead; use these for fine-grained or custom policies.`,
	}

	cmd.AddCommand(&cobra.Command{
		Use:     "get <storage-slug> <bucket-slug>",
		Short:   "Print the bucket's S3 policy (JSON; -o yaml for YAML)",
		Args:    exactArgs(2),
		Example: `  zcp object-storage bucket policy get my-store-1 my-bucket`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			policy, err := svc.GetBucketPolicy(ctx, args[0], args[1])
			if err != nil {
				return fmt.Errorf("object-storage bucket policy get: %w", err)
			}
			if policy == "" {
				fmt.Fprintln(os.Stderr, "(no bucket policy set)")
				return nil
			}
			return printRawDocument(printer, policy)
		},
	})

	var policyFile string
	setCmd := &cobra.Command{
		Use:   "set <storage-slug> <bucket-slug>",
		Short: "Set the bucket's S3 policy from a JSON file (or - for stdin)",
		Args:  exactArgs(2),
		Example: `  zcp object-storage bucket policy set my-store-1 my-bucket --file policy.json
  cat policy.json | zcp object-storage bucket policy set my-store-1 my-bucket --file -`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if policyFile == "" {
				return fmt.Errorf("--file is required (path to a JSON policy, or - for stdin)")
			}
			var data []byte
			var rerr error
			if policyFile == "-" {
				data, rerr = io.ReadAll(os.Stdin)
			} else {
				data, rerr = os.ReadFile(policyFile)
			}
			if rerr != nil {
				return fmt.Errorf("reading policy: %w", rerr)
			}
			if !json.Valid(data) {
				return fmt.Errorf("policy is not valid JSON")
			}
			_, client, _, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			if err := svc.PutBucketPolicy(ctx, args[0], args[1], string(data)); err != nil {
				return fmt.Errorf("object-storage bucket policy set: %w", err)
			}
			fmt.Fprintf(os.Stdout, "Policy set on bucket %q.\n", args[1])
			return nil
		},
	}
	setCmd.Flags().StringVar(&policyFile, "file", "", "Path to a JSON policy file, or - to read from stdin (required)")
	cmd.AddCommand(setCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "delete <storage-slug> <bucket-slug>",
		Short: "Remove the bucket's S3 policy",
		Args:  exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, _, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			if err := svc.PutBucketPolicy(ctx, args[0], args[1], ""); err != nil {
				return fmt.Errorf("object-storage bucket policy delete: %w", err)
			}
			fmt.Fprintf(os.Stdout, "Policy removed from bucket %q.\n", args[1])
			return nil
		},
	})
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
	cmd.AddCommand(newOSObjectDownloadCmd())
	cmd.AddCommand(newOSObjectURLCmd())
	cmd.AddCommand(newOSObjectPutURLCmd())
	cmd.AddCommand(newOSObjectStatCmd())
	cmd.AddCommand(newOSObjectVersionsCmd())
	cmd.AddCommand(newOSObjectRestoreCmd())
	cmd.AddCommand(newOSObjectCopyCmd())
	cmd.AddCommand(newOSObjectMoveCmd())
	cmd.AddCommand(newOSObjectTagCmd())
	cmd.AddCommand(newOSObjectDeleteCmd())
	return cmd
}

func newOSBucketListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <storage-slug>",
		Short: "List buckets in an object storage instance",
		Args:  exactArgs(1),
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
		Args:    exactArgs(2),
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
		Args:    exactArgs(1),
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
	var purge bool
	cmd := &cobra.Command{
		Use:   "delete <storage-slug> <bucket-slug>",
		Short: "Delete a bucket from an object storage instance",
		Args:  exactArgs(2),
		Example: `  zcp object-storage bucket delete my-storage-1 my-bucket
  zcp object-storage bucket delete my-storage-1 my-bucket --purge -y`,
		Long: `Delete a bucket. The bucket must be empty; a bucket that has ever had
versioning enabled retains object versions/delete-markers that block deletion —
pass --purge to remove all objects and versions first.`,
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

			if purge {
				n, perr := svc.EmptyBucket(ctx, storageSlug, bucketSlug)
				if perr != nil {
					return fmt.Errorf("object-storage bucket delete --purge: %w", perr)
				}
				fmt.Fprintf(os.Stderr, "Purged %d object/version entries from %q.\n", n, bucketSlug)
			}

			if err := svc.DeleteBucket(ctx, storageSlug, bucketSlug); err != nil {
				if apierrors.IsResourceNotFound(err) {
					fmt.Fprintf(os.Stderr, "Bucket %q not found — already deleted.\n", bucketSlug)
					return nil
				}
				return fmt.Errorf("object-storage bucket delete: %w", err)
			}
			printer.Fprintf("Bucket %q deleted from %q.\n", bucketSlug, storageSlug)
			return nil
		},
	}
	cmd.Flags().BoolVar(&purge, "purge", false, "Empty the bucket (all objects + versions) before deleting")
	return cmd
}

func newOSBucketSetACLCmd() *cobra.Command {
	var acl string

	cmd := &cobra.Command{
		Use:   "set-acl <storage-slug> <bucket-slug>",
		Short: "Set bucket visibility (public/private) via an S3 bucket policy",
		Args:  exactArgs(2),
		Example: `  zcp object-storage bucket set-acl my-storage-1 my-bucket --acl public-read
  zcp object-storage bucket set-acl my-storage-1 my-bucket --acl private`,
		Long: `Set whether a bucket's objects are anonymously accessible.

This applies an S3 bucket policy (the mechanism that grants anonymous access;
a bucket canned ACL does not grant object GET):
  private            objects are not publicly accessible (removes the policy)
  public-read        anyone can list the bucket and download objects
  public-read-write  anyone can also upload and delete objects (use with care)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			switch acl {
			case "private", "public-read", "public-read-write":
			case "":
				return fmt.Errorf("--acl is required (values: private, public-read, public-read-write)")
			default:
				return fmt.Errorf("unsupported --acl %q (values: private, public-read, public-read-write)", acl)
			}

			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			if err := svc.SetBucketVisibility(ctx, args[0], args[1], acl); err != nil {
				return fmt.Errorf("object-storage bucket set-acl: %w", err)
			}
			return printer.PrintTable(
				[]string{"BUCKET", "VISIBILITY"},
				[][]string{{args[1], acl}},
			)
		},
	}
	cmd.Flags().StringVar(&acl, "acl", "", "Visibility: private, public-read, or public-read-write (required)")
	return cmd
}

func newOSObjectListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <storage-slug> <bucket-slug>",
		Short: "List objects in a bucket",
		Args:  exactArgs(2),
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
		Args:    exactArgs(3),
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
		metadata    []string
	)

	cmd := &cobra.Command{
		Use:   "put <storage-slug> <bucket-slug> <local-file>",
		Short: "Upload a local file to a bucket",
		Args:  exactArgs(3),
		Example: `  zcp object-storage object put my-storage-1 my-bucket ./report.pdf
  zcp object-storage object put my-storage-1 my-bucket ./logo.png --key images/logo.png
  zcp object-storage object put my-storage-1 my-bucket ./data.bin --content-type application/octet-stream --metadata owner=alice`,
		RunE: func(cmd *cobra.Command, args []string) error {
			storageSlug, bucketSlug, localFile := args[0], args[1], args[2]

			meta, err := parseKVTags(metadata)
			if err != nil {
				return err
			}

			_, client, _, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			size, err := svc.PutObject(ctx, storageSlug, bucketSlug, localFile, key, contentType, meta)
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
	cmd.Flags().StringArrayVar(&metadata, "metadata", nil, "User metadata as key=value (x-amz-meta-*, repeatable)")
	return cmd
}

func newOSObjectDownloadCmd() *cobra.Command {
	var dest, versionID string

	cmd := &cobra.Command{
		Use:   "download <storage-slug> <bucket-slug> <object-key>",
		Short: "Download an object from a bucket to a local file",
		Args:  exactArgs(3),
		Example: `  zcp object-storage object download my-storage-1 my-bucket report.pdf
  zcp object-storage object download my-storage-1 my-bucket images/logo.png --dest ./logo.png
  zcp object-storage object download my-storage-1 my-bucket report.pdf --version-id <id>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			storageSlug, bucketSlug, objectKey := args[0], args[1], args[2]

			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			path, size, err := svc.DownloadObject(ctx, storageSlug, bucketSlug, objectKey, dest, versionID)
			if err != nil {
				return fmt.Errorf("object-storage object download: %w", err)
			}
			return printer.PrintTable(
				[]string{"BUCKET", "KEY", "PATH", "BYTES"},
				[][]string{{bucketSlug, objectKey, path, fmt.Sprintf("%d", size)}},
			)
		},
	}
	cmd.Flags().StringVar(&dest, "dest", "", "Local destination file or directory (defaults to the object's base name in the current directory)")
	cmd.Flags().StringVar(&versionID, "version-id", "", "Download a specific object version")
	return cmd
}

func newOSObjectURLCmd() *cobra.Command {
	var expires time.Duration

	cmd := &cobra.Command{
		Use:   "url <storage-slug> <bucket-slug> <object-key>",
		Short: "Generate a pre-signed, time-limited URL a client can use to download an object",
		Args:  exactArgs(3),
		Example: `  zcp object-storage object url my-storage-1 my-bucket report.pdf
  zcp object-storage object url my-storage-1 my-bucket report.pdf --expires 24h`,
		Long: `Generate a pre-signed HTTPS URL for an object. Anyone with the URL can download
the object until it expires — no ZCP credentials needed — even if the bucket is
private. Maximum lifetime is 7 days (S3 signature limit).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if expires <= 0 || expires > 7*24*time.Hour {
				return fmt.Errorf("--expires must be between 1s and 168h (7 days)")
			}
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			u, err := svc.PresignObjectURL(ctx, args[0], args[1], args[2], expires)
			if err != nil {
				return fmt.Errorf("object-storage object url: %w", err)
			}
			// Default to the bare URL (easy to pipe/$()-capture); honor -o json/yaml
			// via the resolved printer format (the single source of truth, which
			// already folds in ZCP_OUTPUT).
			if f := printer.Format(); f == output.FormatJSON || f == output.FormatYAML {
				return printer.PrintTable([]string{"URL"}, [][]string{{u}})
			}
			fmt.Fprintln(os.Stdout, u)
			return nil
		},
	}
	cmd.Flags().DurationVar(&expires, "expires", time.Hour, "URL lifetime (e.g. 30m, 24h; max 168h)")
	return cmd
}

func newOSObjectPutURLCmd() *cobra.Command {
	var expires time.Duration
	cmd := &cobra.Command{
		Use:   "put-url <storage-slug> <bucket-slug> <object-key>",
		Short: "Generate a pre-signed URL a client can use to UPLOAD an object (HTTP PUT)",
		Args:  exactArgs(3),
		Example: `  zcp object-storage object put-url my-store my-bucket upload.bin --expires 30m
  # then: curl -T ./file.bin "<url>"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if expires <= 0 || expires > 7*24*time.Hour {
				return fmt.Errorf("--expires must be between 1s and 168h (7 days)")
			}
			_, client, _, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			u, err := svc.PresignPutURL(ctx, args[0], args[1], args[2], expires)
			if err != nil {
				return fmt.Errorf("object-storage object put-url: %w", err)
			}
			fmt.Fprintln(os.Stdout, u)
			return nil
		},
	}
	cmd.Flags().DurationVar(&expires, "expires", time.Hour, "URL lifetime (e.g. 30m, 24h; max 168h)")
	return cmd
}

func newOSObjectStatCmd() *cobra.Command {
	var versionID string
	cmd := &cobra.Command{
		Use:   "stat <storage-slug> <bucket-slug> <object-key>",
		Short: "Show full S3 metadata for an object (size, content-type, ETag, user metadata)",
		Args:  exactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			st, err := svc.StatObject(ctx, args[0], args[1], args[2], versionID)
			if err != nil {
				return fmt.Errorf("object-storage object stat: %w", err)
			}
			rows := [][]string{
				{"Key", st.Key},
				{"Size", fmt.Sprintf("%d", st.Size)},
				{"Content-Type", st.ContentType},
				{"ETag", st.ETag},
				{"Storage Class", st.StorageClass},
				{"Version ID", st.VersionID},
				{"Last Modified", st.LastModified.Format(time.RFC3339)},
			}
			for k, v := range st.UserMetadata {
				rows = append(rows, []string{"meta:" + k, v})
			}
			return printer.PrintTable([]string{"FIELD", "VALUE"}, rows)
		},
	}
	cmd.Flags().StringVar(&versionID, "version-id", "", "Stat a specific object version")
	return cmd
}

func newOSObjectVersionsCmd() *cobra.Command {
	var prefix string
	cmd := &cobra.Command{
		Use:   "versions <storage-slug> <bucket-slug>",
		Short: "List object versions and delete markers (requires versioning)",
		Args:  exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			vs, err := svc.ListObjectVersions(ctx, args[0], args[1], prefix)
			if err != nil {
				return fmt.Errorf("object-storage object versions: %w", err)
			}
			rows := make([][]string, 0, len(vs))
			for _, v := range vs {
				kind := "version"
				if v.IsDeleteMarker {
					kind = "delete-marker"
				}
				rows = append(rows, []string{
					v.Key, v.VersionID, kind,
					strconv.FormatBool(v.IsLatest),
					fmt.Sprintf("%d", v.Size),
					v.LastModified.Format(time.RFC3339),
				})
			}
			return printer.PrintTable([]string{"KEY", "VERSION ID", "TYPE", "LATEST", "SIZE", "MODIFIED"}, rows)
		},
	}
	cmd.Flags().StringVar(&prefix, "prefix", "", "Only list versions under this key prefix")
	return cmd
}

func newOSObjectRestoreCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restore <storage-slug> <bucket-slug> <object-key>",
		Short: "Undelete an object by removing its latest delete marker",
		Args:  exactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, _, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			vid, err := svc.RestoreObject(ctx, args[0], args[1], args[2])
			if err != nil {
				return fmt.Errorf("object-storage object restore: %w", err)
			}
			fmt.Fprintf(os.Stdout, "Restored %q (removed delete marker %s).\n", args[2], vid)
			return nil
		},
	}
}

func newOSObjectCopyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "copy <storage-slug> <src-bucket> <src-key> <dst-bucket> <dst-key>",
		Short: "Server-side copy an object (no download/upload round-trip)",
		Args:  exactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, _, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			if err := svc.CopyObject(ctx, args[0], args[1], args[2], args[3], args[4]); err != nil {
				return fmt.Errorf("object-storage object copy: %w", err)
			}
			fmt.Fprintf(os.Stdout, "Copied %s/%s to %s/%s.\n", args[1], args[2], args[3], args[4])
			return nil
		},
	}
}

func newOSObjectMoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "move <storage-slug> <src-bucket> <src-key> <dst-bucket> <dst-key>",
		Short: "Server-side move an object (copy then delete the source)",
		Args:  exactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, _, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			if err := svc.MoveObject(ctx, args[0], args[1], args[2], args[3], args[4]); err != nil {
				return fmt.Errorf("object-storage object move: %w", err)
			}
			fmt.Fprintf(os.Stdout, "Moved %s/%s to %s/%s.\n", args[1], args[2], args[3], args[4])
			return nil
		},
	}
}

func newOSObjectTagCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "tag", Short: "Manage object tags"}
	cmd.AddCommand(&cobra.Command{
		Use:   "get <storage-slug> <bucket-slug> <object-key>",
		Short: "Show an object's tags",
		Args:  exactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			m, err := svc.GetObjectTags(ctx, args[0], args[1], args[2])
			if err != nil {
				return fmt.Errorf("object-storage object tag get: %w", err)
			}
			return printer.PrintTable([]string{"KEY", "VALUE"}, tagRows(m))
		},
	})
	var setTags []string
	setCmd := &cobra.Command{
		Use:   "set <storage-slug> <bucket-slug> <object-key>",
		Short: "Replace an object's tags",
		Args:  exactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := parseKVTags(setTags)
			if err != nil {
				return err
			}
			if len(m) == 0 {
				return fmt.Errorf("at least one --tag key=value is required")
			}
			_, client, _, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			if err := svc.SetObjectTags(ctx, args[0], args[1], args[2], m); err != nil {
				return fmt.Errorf("object-storage object tag set: %w", err)
			}
			fmt.Fprintf(os.Stdout, "Set %d tag(s) on %s/%s.\n", len(m), args[1], args[2])
			return nil
		},
	}
	setCmd.Flags().StringArrayVar(&setTags, "tag", nil, "Tag as key=value (repeatable)")
	cmd.AddCommand(setCmd)
	cmd.AddCommand(&cobra.Command{
		Use:   "delete <storage-slug> <bucket-slug> <object-key>",
		Short: "Remove all tags from an object",
		Args:  exactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, _, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := objectstorage.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			if err := svc.DeleteObjectTags(ctx, args[0], args[1], args[2]); err != nil {
				return fmt.Errorf("object-storage object tag delete: %w", err)
			}
			fmt.Fprintf(os.Stdout, "Tags removed from %s/%s.\n", args[1], args[2])
			return nil
		},
	})
	return cmd
}

func newOSObjectDeleteCmd() *cobra.Command {
	var versionID string
	cmd := &cobra.Command{
		Use:   "delete <storage-slug> <bucket-slug> <object-key>",
		Short: "Delete an object from a bucket (or a specific version with --version-id)",
		Args:  exactArgs(3),
		Example: `  zcp object-storage object delete my-storage-1 my-bucket report.pdf
  zcp object-storage object delete my-storage-1 my-bucket report.pdf --version-id <id> -y`,
		RunE: func(cmd *cobra.Command, args []string) error {
			storageSlug, bucketSlug, objectKey := args[0], args[1], args[2]

			if !autoApproved(cmd) {
				target := objectKey
				if versionID != "" {
					target = objectKey + " (version " + versionID + ")"
				}
				fmt.Fprintf(os.Stderr, "Delete object %q from bucket %q? [y/N]: ", target, bucketSlug)
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

			if err := svc.DeleteObject(ctx, storageSlug, bucketSlug, objectKey, versionID); err != nil {
				if apierrors.IsResourceNotFound(err) {
					fmt.Fprintf(os.Stderr, "Object %q not found — already deleted.\n", objectKey)
					return nil
				}
				return fmt.Errorf("object-storage object delete: %w", err)
			}

			fmt.Fprintf(os.Stdout, "Deleted %q from bucket %q.\n", objectKey, bucketSlug)
			return nil
		},
	}
	cmd.Flags().StringVar(&versionID, "version-id", "", "Delete a specific object version (instead of adding a delete marker)")
	return cmd
}
