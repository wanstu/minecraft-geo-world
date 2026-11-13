#!/usr/bin/env python3
"""Real, persisted Minecraft voxel acceptance for R8.1p (v1.1.29).

Delegates to the locked R8.1e independent Anvil reader. Requires every
sampled column to be generated; empty chunks cannot pass as flat terrain.
"""
import r81e_multiseed_actual_voxel_audit as audit

audit.RULES = "geoworld-rules-v1.1.29"
audit.OFFSETS = list(range(-104, 105, 8))

if __name__ == "__main__":
    audit.main()
