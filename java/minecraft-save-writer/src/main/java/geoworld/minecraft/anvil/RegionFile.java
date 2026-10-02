package geoworld.minecraft.anvil;

import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.nio.file.AtomicMoveNotSupportedException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.StandardCopyOption;
import java.time.Instant;
import java.util.Arrays;
import java.util.Collection;
import java.util.zip.GZIPInputStream;
import java.util.zip.InflaterInputStream;
import java.util.zip.DeflaterOutputStream;

public final class RegionFile {
    public static final int CHUNKS_PER_AXIS = 32;
    public static final int CHUNK_SLOTS = CHUNKS_PER_AXIS * CHUNKS_PER_AXIS;
    public static final int SECTOR_BYTES = 4096;
    public static final int HEADER_SECTORS = 2;
    public static final int HEADER_BYTES = HEADER_SECTORS * SECTOR_BYTES;

    private RegionFile() {}

    public record Chunk(int chunkX, int chunkZ, byte[] nbt) {
        public Chunk {
            if (nbt == null) throw new NullPointerException("chunk NBT");
            nbt = nbt.clone();
        }

        @Override
        public byte[] nbt() {
            return nbt.clone();
        }

        byte[] rawNbt() {
            return nbt;
        }
    }

    public record Location(int sectorOffset, int sectorCount, int timestampSeconds) {}

    private record EncodedChunk(
        int slot,
        byte[] compressed,
        int sectorCount
    ) {}

    public static int regionXForChunk(int chunkX) {
        return Math.floorDiv(chunkX, CHUNKS_PER_AXIS);
    }

    public static int regionZForChunk(int chunkZ) {
        return Math.floorDiv(chunkZ, CHUNKS_PER_AXIS);
    }

    public static int localX(int chunkX) {
        return Math.floorMod(chunkX, CHUNKS_PER_AXIS);
    }

    public static int localZ(int chunkZ) {
        return Math.floorMod(chunkZ, CHUNKS_PER_AXIS);
    }

    public static Path regionPath(Path regionDirectory, int regionX, int regionZ) {
        return regionDirectory.resolve("r." + regionX + "." + regionZ + ".mca");
    }

    public static void write(
        Path path,
        int regionX,
        int regionZ,
        Collection<Chunk> chunks
    ) throws IOException {
        long now = Instant.now().getEpochSecond();
        write(path, regionX, regionZ, chunks, (int) Math.min(Integer.MAX_VALUE, now));
    }

    public static void write(
        Path path,
        int regionX,
        int regionZ,
        Collection<Chunk> chunks,
        int timestampSeconds
    ) throws IOException {
        if (path == null || chunks == null) {
            throw new NullPointerException("region path/chunks");
        }

        EncodedChunk[] slots = new EncodedChunk[CHUNK_SLOTS];
        int dataSectors = 0;

        for (Chunk chunk : chunks) {
            if (
                regionXForChunk(chunk.chunkX()) != regionX ||
                regionZForChunk(chunk.chunkZ()) != regionZ
            ) {
                throw new IllegalArgumentException(
                    "chunk " + chunk.chunkX() + "," + chunk.chunkZ() +
                    " does not belong to region " + regionX + "," + regionZ
                );
            }

            int slot = localX(chunk.chunkX()) + localZ(chunk.chunkZ()) * CHUNKS_PER_AXIS;
            if (slots[slot] != null) {
                throw new IllegalArgumentException(
                    "duplicate chunk slot " + chunk.chunkX() + "," + chunk.chunkZ()
                );
            }

            byte[] compressed = compressZlib(chunk.rawNbt());
            int recordBytes = 4 + 1 + compressed.length;
            int sectors = ceilDiv(recordBytes, SECTOR_BYTES);
            if (sectors <= 0 || sectors > 255) {
                throw new IllegalArgumentException(
                    "chunk " + chunk.chunkX() + "," + chunk.chunkZ() +
                    " requires " + sectors +
                    " sectors; external .mcc chunks are not implemented"
                );
            }
            dataSectors = Math.addExact(dataSectors, sectors);
            slots[slot] = new EncodedChunk(slot, compressed, sectors);
        }

        int totalSectors = Math.addExact(HEADER_SECTORS, dataSectors);
        if (totalSectors > 0xFFFFFF) {
            throw new IllegalArgumentException("region sector offset exceeds 24-bit header limit");
        }

        byte[] file = new byte[Math.multiplyExact(totalSectors, SECTOR_BYTES)];
        int nextSector = HEADER_SECTORS;

        for (int slot = 0; slot < CHUNK_SLOTS; slot++) {
            EncodedChunk encoded = slots[slot];
            if (encoded == null) continue;

            int locationOffset = slot * 4;
            file[locationOffset] = (byte) ((nextSector >>> 16) & 0xff);
            file[locationOffset + 1] = (byte) ((nextSector >>> 8) & 0xff);
            file[locationOffset + 2] = (byte) (nextSector & 0xff);
            file[locationOffset + 3] = (byte) encoded.sectorCount();

            putInt(file, SECTOR_BYTES + slot * 4, timestampSeconds);

            int start = Math.multiplyExact(nextSector, SECTOR_BYTES);
            putInt(file, start, encoded.compressed().length + 1);
            file[start + 4] = 2; // zlib
            System.arraycopy(
                encoded.compressed(),
                0,
                file,
                start + 5,
                encoded.compressed().length
            );
            nextSector += encoded.sectorCount();
        }

        Path parent = path.toAbsolutePath().getParent();
        if (parent != null) Files.createDirectories(parent);
        Path temp = Files.createTempFile(parent, "." + path.getFileName(), ".tmp");
        boolean published = false;
        try {
            Files.write(temp, file);
            try {
                Files.move(
                    temp,
                    path,
                    StandardCopyOption.ATOMIC_MOVE,
                    StandardCopyOption.REPLACE_EXISTING
                );
            } catch (AtomicMoveNotSupportedException e) {
                Files.move(temp, path, StandardCopyOption.REPLACE_EXISTING);
            }
            published = true;
        } finally {
            if (!published) Files.deleteIfExists(temp);
        }
    }

    public static Location location(Path path, int chunkX, int chunkZ) throws IOException {
        byte[] file = Files.readAllBytes(path);
        requireHeader(file);
        int slot = localX(chunkX) + localZ(chunkZ) * CHUNKS_PER_AXIS;
        int p = slot * 4;
        int offset =
            (Byte.toUnsignedInt(file[p]) << 16) |
            (Byte.toUnsignedInt(file[p + 1]) << 8) |
            Byte.toUnsignedInt(file[p + 2]);
        int sectors = Byte.toUnsignedInt(file[p + 3]);
        int timestamp = getInt(file, SECTOR_BYTES + slot * 4);
        return new Location(offset, sectors, timestamp);
    }

    public static byte[] readChunkNbt(Path path, int chunkX, int chunkZ) throws IOException {
        byte[] file = Files.readAllBytes(path);
        requireHeader(file);

        int slot = localX(chunkX) + localZ(chunkZ) * CHUNKS_PER_AXIS;
        int p = slot * 4;
        int sectorOffset =
            (Byte.toUnsignedInt(file[p]) << 16) |
            (Byte.toUnsignedInt(file[p + 1]) << 8) |
            Byte.toUnsignedInt(file[p + 2]);
        int sectorCount = Byte.toUnsignedInt(file[p + 3]);

        if (sectorOffset == 0 && sectorCount == 0) return null;
        if (sectorOffset < HEADER_SECTORS || sectorCount <= 0) {
            throw new IOException("invalid region location entry");
        }

        long startLong = (long) sectorOffset * SECTOR_BYTES;
        long allocationEndLong = startLong + (long) sectorCount * SECTOR_BYTES;
        if (
            startLong > Integer.MAX_VALUE ||
            allocationEndLong > file.length ||
            allocationEndLong > Integer.MAX_VALUE
        ) {
            throw new IOException("chunk allocation lies outside region file");
        }

        int start = (int) startLong;
        int length = getInt(file, start);
        if (length < 1 || length > sectorCount * SECTOR_BYTES - 4) {
            throw new IOException("invalid chunk record length");
        }
        if ((long) start + 4L + length > file.length) {
            throw new IOException("truncated chunk record");
        }

        int compression = Byte.toUnsignedInt(file[start + 4]);
        if ((compression & 0x80) != 0) {
            throw new IOException("external .mcc chunk records are not supported");
        }

        byte[] payload = Arrays.copyOfRange(file, start + 5, start + 4 + length);
        return switch (compression) {
            case 1 -> decompress(new GZIPInputStream(new ByteArrayInputStream(payload)));
            case 2 -> decompress(new InflaterInputStream(new ByteArrayInputStream(payload)));
            case 3 -> payload;
            default -> throw new IOException("unsupported chunk compression type: " + compression);
        };
    }

    private static byte[] compressZlib(byte[] input) throws IOException {
        ByteArrayOutputStream out = new ByteArrayOutputStream();
        try (DeflaterOutputStream deflater = new DeflaterOutputStream(out)) {
            deflater.write(input);
        }
        return out.toByteArray();
    }

    private static byte[] decompress(InputStream input) throws IOException {
        try (InputStream in = input; ByteArrayOutputStream out = new ByteArrayOutputStream()) {
            in.transferTo(out);
            return out.toByteArray();
        }
    }

    private static void requireHeader(byte[] file) throws IOException {
        if (file.length < HEADER_BYTES || file.length % SECTOR_BYTES != 0) {
            throw new IOException("invalid Anvil region file length");
        }
    }

    private static int ceilDiv(int value, int divisor) {
        return (value + divisor - 1) / divisor;
    }

    private static void putInt(byte[] data, int offset, int value) {
        data[offset] = (byte) (value >>> 24);
        data[offset + 1] = (byte) (value >>> 16);
        data[offset + 2] = (byte) (value >>> 8);
        data[offset + 3] = (byte) value;
    }

    private static int getInt(byte[] data, int offset) {
        return
            (Byte.toUnsignedInt(data[offset]) << 24) |
            (Byte.toUnsignedInt(data[offset + 1]) << 16) |
            (Byte.toUnsignedInt(data[offset + 2]) << 8) |
            Byte.toUnsignedInt(data[offset + 3]);
    }
}
