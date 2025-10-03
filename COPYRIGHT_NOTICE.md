# Copyright and License Attribution

## OpenTelemetry Integration

The OpenTelemetry integration feature was contributed by Robert B Gordon and is covered under the Apache License 2.0, consistent with the rest of the stern project.

### New Files (Copyright 2025 Robert B Gordon)

All files in the `stern/otel/` directory are original works:

- `stern/otel/exporter.go`
- `stern/otel/transformer.go`
- `stern/otel/resource.go`
- `stern/otel/transformer_test.go`
- `stern/otel/resource_test.go`
- `stern/otel/README.md`

**Copyright**: 2025 Robert B Gordon <rbg@openrbg.com>
**License**: Apache License 2.0

### Modified Files (Derivative Works)

The following existing files were modified to add OpenTelemetry support. These files maintain their original copyright while noting the modifications:

- `stern/config.go` - Added OTel configuration fields
- `stern/tail.go` - Added OTel log emission
- `stern/stern.go` - Added graceful shutdown for OTel
- `cmd/cmd.go` - Added CLI flags and initialization

**Original Copyright**: 2016 Wercker Holding BV
**Modifications Copyright**: 2025 Robert B Gordon <rbg@openrbg.com>
**License**: Apache License 2.0 (unchanged)

## License Compliance

All code contributions follow the Apache License 2.0 terms:

1. **Same License**: All modifications use Apache 2.0, matching the original project
2. **Attribution**: Copyright notices properly attribute both original and modification authors
3. **Patent Grant**: Apache 2.0 patent grant applies to all contributions
4. **Derivative Works**: Modified files clearly indicate changes with copyright notices

## Attribution Format

### New Files
```go
//   Copyright 2025 Robert B Gordon <rbg@openrbg.com>
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   ...
```

### Modified Files
```go
//   Copyright 2016 Wercker Holding BV
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   ...
//
//   Modifications for OpenTelemetry support:
//   Copyright 2025 Robert B Gordon <rbg@openrbg.com>
```

## Legal Notes

1. **License Compatibility**: Apache 2.0 is compatible with itself - no license conflicts
2. **Contribution**: This contribution can be merged into the upstream stern project
3. **Rights**: All contributors retain copyright while granting usage rights per Apache 2.0
4. **No Warranty**: As per Apache 2.0, all code is provided "AS IS" without warranty

## Contact

**Contributor**: Robert B Gordon
**Email**: rbg@openrbg.com
**Contribution Date**: October 2025
**Feature**: OpenTelemetry OTLP log export integration
