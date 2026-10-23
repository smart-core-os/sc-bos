# proto

API definitions for sc-bos. Each API lives in `smartcore/bos/<name>/v1/` with
`package smartcore.bos.<name>.v1` and `option go_package = "github.com/smart-core-os/sc-bos/pkg/proto/<name>pb"`.
The trait's history aspect, if any, goes in a sibling `<name>_history.proto`.

## Designing a trait

- **Name the trait after the state it manages, as a noun**: `Boot`, not `Reboot`. Actions are RPCs on the API
  (`BootApi.Reboot`).
- **Services are aspects**: `<Trait>Api` for the resource (`Get<Resource>`, `Pull<Resource>`, `Update<Resource>`),
  `<Trait>Info` for metadata, `<Trait>History` for past values.
- **Every request has `string name = 1`** identifying the target. `genproto` only generates a router for a service
  when all its requests do.
- **Reads take a `google.protobuf.FieldMask read_mask`**; updates take an `update_mask`.
- **History messages are `<Name>Record`**, listed by `List<Name>Records` returning `repeated <Name>Record
  <snake_name>_records`, each with a `record_time`.
- **Don't add fields a client can derive** from other fields (`uptime` when `boot_time` is present). They go stale in
  streams and make the API chattier.
- **Comment every field** with its meaning, units, and what an unset value means.

## After changing a .proto

1. `bash scripts/gen-proto.sh` (needs the pinned toolchain; see the root [AGENTS.md](../AGENTS.md)). Commit
   `pkg/proto/<name>pb` and `ui/ui-gen/proto` with the change. CI's proto-check fails if they are stale.
2. For a new trait, register its services in `pkg/node/alltraits/services.go`.
3. Authorisation for standard verbs (`Get`, `Pull`, `List`, `Update`, ...) comes from
   `pkg/auth/policy/default/smartcore.rego`. Only RPCs that don't fit those verbs need their own
   `smartcore.bos.<name>.v1.<Service>.rego` there (see `smartcore.bos.boot.v1.BootApi.rego`).
