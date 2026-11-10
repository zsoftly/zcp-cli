# zcp v1.3.1 Release Notes

zcp v1.3.1 fixes network ACL listing for VPCs with more than one page of ACLs.

## Highlights

**Every network ACL in a VPC is visible.** The ACL list API returns ACLs in
pages, 10 by default. zcp read only the first page, so once a VPC had more ACLs
than that, its oldest ACLs disappeared from `zcp acl list` and could not be
found by name or ID, although they still existed and still applied to their
networks. zcp now reads every page.

```bash
# Lists all ACLs in the VPC, including the oldest.
zcp acl list <vpc-slug>

# Works for any ACL in the VPC, not only those on the first page.
zcp acl rules <vpc-slug> <acl-name-or-id>
```

## Fixed

- `zcp acl list` shows every ACL in the VPC, not only the first page.
- `zcp acl rules`, `zcp acl delete` and `zcp acl replace --vpc` find ACLs that
  are not on the first page, instead of failing with
  "ACL ... not found in VPC".
- Go callers of `acl.Service.List`, `acl.Service.Resolve` and
  `vpc.Service.ListACLs` receive the complete ACL list. Tools built on the zcp
  module, such as the ZCP Terraform provider, need a release that picks up this
  version to benefit.

Reported in [#131](https://github.com/zsoftly/zcp-cli/issues/131).
