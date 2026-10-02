package geoworld.minecraft;

import geoworld.minecraft.anvil.RegionFile;
import geoworld.minecraft.nbt.Nbt;

import java.nio.file.Files;
import java.nio.file.Path;
import java.util.Arrays;
import java.util.List;

public final class StorageVerifier {
    private StorageVerifier() {}

    public static void main(String[] args) throws Exception {
        verifyKnownNbtBytes();
        verifyUtf8StringBytes();
        verifyRegionFile();
        verifyNegativeChunkMapping();
        System.out.println("Minecraft storage verification passed.");
    }

    private static void verifyKnownNbtBytes() {
        Nbt.CompoundTag root = Nbt.compound()
            .put("A", Nbt.byteTag(1));

        byte[] got = Nbt.writeRoot("", root);
        byte[] want = new byte[] {
            0x0A,
            0x00, 0x00,
            0x01,
            0x00, 0x01, 0x41,
            0x01,
            0x00
        };

        requireBytes("known minimal NBT", got, want);
        System.out.println("PASS known minimal NBT byte layout");
    }

    private static void verifyUtf8StringBytes() {
        Nbt.CompoundTag root = Nbt.compound()
            .put("s", Nbt.string("名"));

        byte[] got = Nbt.writeRoot("", root);
        byte[] want = new byte[] {
            0x0A,
            0x00, 0x00,
            0x08,
            0x00, 0x01, 0x73,
            0x00, 0x03,
            (byte) 0xE5, (byte) 0x90, (byte) 0x8D,
            0x00
        };

        requireBytes("UTF-8 NBT string", got, want);
        System.out.println("PASS NBT strings use standard UTF-8 byte length");
    }

    private static void verifyRegionFile() throws Exception {
        Path dir = Files.createTempDirectory("geoworld-region-test-");
        try {
            Path path = RegionFile.regionPath(dir, 0, 0);

            byte[] first = Nbt.writeRoot(
                "",
                Nbt.compound()
                    .put("xPos", Nbt.intTag(0))
                    .put("zPos", Nbt.intTag(0))
                    .put("Status", Nbt.string("geoworld:test"))
            );
            byte[] last = Nbt.writeRoot(
                "",
                Nbt.compound()
                    .put("xPos", Nbt.intTag(31))
                    .put("zPos", Nbt.intTag(31))
                    .put("Name", Nbt.string("边界"))
            );

            int timestamp = 123456789;
            RegionFile.write(
                path,
                0,
                0,
                List.of(
                    new RegionFile.Chunk(0, 0, first),
                    new RegionFile.Chunk(31, 31, last)
                ),
                timestamp
            );

            RegionFile.Location firstLocation = RegionFile.location(path, 0, 0);
            if (
                firstLocation.sectorOffset() != 2 ||
                firstLocation.sectorCount() != 1 ||
                firstLocation.timestampSeconds() != timestamp
            ) {
                throw new AssertionError("unexpected first chunk location: " + firstLocation);
            }

            RegionFile.Location lastLocation = RegionFile.location(path, 31, 31);
            if (
                lastLocation.sectorOffset() != 3 ||
                lastLocation.sectorCount() != 1 ||
                lastLocation.timestampSeconds() != timestamp
            ) {
                throw new AssertionError("unexpected last chunk location: " + lastLocation);
            }

            requireBytes(
                "region first chunk round trip",
                RegionFile.readChunkNbt(path, 0, 0),
                first
            );
            requireBytes(
                "region last chunk round trip",
                RegionFile.readChunkNbt(path, 31, 31),
                last
            );

            if (RegionFile.readChunkNbt(path, 1, 1) != null) {
                throw new AssertionError("empty region slot unexpectedly contains a chunk");
            }

            long expectedBytes = 4L * RegionFile.SECTOR_BYTES;
            if (Files.size(path) != expectedBytes) {
                throw new AssertionError(
                    "region file size=" + Files.size(path) + " want=" + expectedBytes
                );
            }

            System.out.println("PASS Anvil header, zlib chunk records and sector allocation");
        } finally {
            deleteTree(dir);
        }
    }

    private static void verifyNegativeChunkMapping() throws Exception {
        if (
            RegionFile.regionXForChunk(-1) != -1 ||
            RegionFile.regionZForChunk(-1) != -1 ||
            RegionFile.localX(-1) != 31 ||
            RegionFile.localZ(-1) != 31
        ) {
            throw new AssertionError("negative chunk floorDiv/floorMod mapping is wrong");
        }

        Path dir = Files.createTempDirectory("geoworld-region-negative-");
        try {
            Path path = RegionFile.regionPath(dir, -1, -1);
            byte[] nbt = Nbt.writeRoot(
                "",
                Nbt.compound()
                    .put("xPos", Nbt.intTag(-1))
                    .put("zPos", Nbt.intTag(-1))
            );

            RegionFile.write(
                path,
                -1,
                -1,
                List.of(new RegionFile.Chunk(-1, -1, nbt)),
                7
            );

            RegionFile.Location location = RegionFile.location(path, -1, -1);
            if (
                location.sectorOffset() != 2 ||
                location.sectorCount() != 1 ||
                location.timestampSeconds() != 7
            ) {
                throw new AssertionError("negative chunk location is wrong: " + location);
            }
            requireBytes(
                "negative chunk region round trip",
                RegionFile.readChunkNbt(path, -1, -1),
                nbt
            );
            System.out.println("PASS negative Chunk -> Region/local coordinate mapping");
        } finally {
            deleteTree(dir);
        }
    }

    private static void requireBytes(String name, byte[] got, byte[] want) {
        if (!Arrays.equals(got, want)) {
            throw new AssertionError(
                name + " mismatch\ngot =" + hex(got) + "\nwant=" + hex(want)
            );
        }
    }

    private static String hex(byte[] value) {
        StringBuilder out = new StringBuilder(value.length * 3);
        for (byte b : value) {
            if (!out.isEmpty()) out.append(' ');
            out.append(String.format("%02X", Byte.toUnsignedInt(b)));
        }
        return out.toString();
    }

    private static void deleteTree(Path root) throws Exception {
        if (!Files.exists(root)) return;
        try (var stream = Files.walk(root)) {
            stream.sorted((a, b) -> b.getNameCount() - a.getNameCount())
                .forEach(path -> {
                    try {
                        Files.deleteIfExists(path);
                    } catch (Exception e) {
                        throw new RuntimeException(e);
                    }
                });
        }
    }
}
