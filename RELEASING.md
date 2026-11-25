# Releasing

The OpenTofu and Terraform registries read the provider from its GitHub
releases: zips of the binary per platform,
`terraform-provider-kayra_<version>_SHA256SUMS`, its GPG signature
`..._SHA256SUMS.sig`, and the manifest
`terraform-provider-kayra_<version>_manifest.json`. GoReleaser makes them
(`.goreleaser.yaml`) in the CI, infrapi/ci's `release: semantic` with
`gpg-sign: true` (`.github/workflows/ci.yml`): on a push to `main`,
go-semantic-release makes the version from the conventional commits, its
tag and the GitHub release, then GoReleaser builds, signs and appends the
files, in the same job.

## Once

1. Create the public repository `infrapi/terraform-provider-kayra` on GitHub
   (the registries want the name `terraform-provider-<name>`) and push
   `main`.
2. Make a GPG key for the releases, RSA 4096, with an expiry date (the
   OpenTofu registry checks it at submission):

   ```bash
   gpg --quick-generate-key "InfraPI releases <releases@infrapi.io>" rsa4096 sign 2y
   gpg --armor --export-secret-keys <fingerprint>   # the GPG_PRIVATE_KEY secret
   gpg --armor --export <fingerprint>               # the public key, for the registries
   ```

3. In the repository's settings, the Actions secrets `GPG_PRIVATE_KEY` and
   `PASSPHRASE`, before a `feat:` or `fix:` reaches `main`: its push
   releases, and the release fails without the key.

## Each release

1. Merge to `main` a `feat:` (a minor version) or a `fix:` (a patch); a
   `chore:` or `docs:` releases nothing. The CI's checks pass, then the
   release. `make snapshot` builds the same files
   locally, unsigned.

## On the OpenTofu registry, once

Submissions go through the registry's GitHub issue forms, in the web UI
only: an issue opened another way is closed.

1. The provider: <https://github.com/opentofu/registry/issues/new?template=provider.yml>,
   repository `infrapi/terraform-provider-kayra`. The registry's automation
   opens the pull request; a maintainer merges it.
2. The GPG key: <https://github.com/opentofu/registry/issues/new?template=provider_key.yml>,
   namespace `infrapi`, the public key. Only a **public** member of the
   `infrapi` GitHub org may submit it.

The registry picks up a new version about 30 minutes after its release.
Then `source = "infrapi/kayra"` resolves to `registry.opentofu.org/infrapi/kayra`.

## On the Terraform registry, once

HashiCorp's registry reads the same GitHub releases; publishing is done in
its web UI, by a member of the `infrapi` GitHub org.

1. The key must be RSA (the registry takes RSA or DSA keys, not ECC): the
   one above.
2. Sign in at <https://registry.terraform.io> with GitHub, and grant the
   registry's GitHub application access to the `infrapi` org (an org owner
   approves it if the org restricts OAuth apps).
3. The signing key: **User Settings > Signing Keys > New GPG Key**,
   namespace `infrapi`, the ASCII-armored public key
   (`gpg --armor --export <fingerprint>`).
4. A signed release must exist (above); check its
   GitHub release has the zips, `..._SHA256SUMS`, `..._SHA256SUMS.sig` and
   `..._manifest.json`.
5. **Publish > Provider**, organization `infrapi`, repository
   `terraform-provider-kayra`, then **Publish**. The registry adds a webhook
   to the repository (it needs admin rights on it): each later release is
   published by itself, its docs from `docs/` at its tag.

Then `source = "infrapi/kayra"` in Terraform resolves to
`registry.terraform.io/infrapi/kayra`, and
<https://registry.terraform.io/providers/infrapi/kayra/latest/docs> shows
`docs/`. A version that fails to publish is listed with its reason in the
provider's settings on the registry (**Resync** retries it).

## Check a release

Both registries list a new version within the hour. From a clean directory:

```bash
cat > main.tf <<'TF'
terraform {
  required_providers {
    kayra = { source = "infrapi/kayra", version = "0.1.0" }
  }
}
TF
tofu init        # registry.opentofu.org/infrapi/kayra
terraform init   # registry.terraform.io/infrapi/kayra, the signature checked
```
