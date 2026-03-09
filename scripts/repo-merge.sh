#!/usr/bin/env bash

# Moves the bos-internal traits into unified packages
go run ./cmd/tools/scfix -only gentraitmv,gentraitref -v
mv pkg/gentrait/historypb/*.proto pkg/proto/historypb/
sed -i '' 's!github.com/smart-core-os/sc-bos/pkg/gentrait/historypb!github.com/smart-core-os/sc-bos/pkg/proto/historypb!g' pkg/proto/historypb/*.proto
sed -i '' 's!smartcore.bos.gentrait.historypb!smartcore.bos.proto.historypb.internal!g' pkg/proto/historypb/*.proto
rm -r pkg/gentrait
go generate ./pkg/proto/historypb