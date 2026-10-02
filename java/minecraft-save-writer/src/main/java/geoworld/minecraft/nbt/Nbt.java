package geoworld.minecraft.nbt;

import java.io.ByteArrayOutputStream;
import java.io.DataOutputStream;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

public final class Nbt {
    private Nbt() {}

    public interface Tag {
        int typeId();
    }

    public record ByteTag(byte value) implements Tag {
        @Override public int typeId() { return 1; }
    }

    public record ShortTag(short value) implements Tag {
        @Override public int typeId() { return 2; }
    }

    public record IntTag(int value) implements Tag {
        @Override public int typeId() { return 3; }
    }

    public record LongTag(long value) implements Tag {
        @Override public int typeId() { return 4; }
    }

    public record FloatTag(float value) implements Tag {
        @Override public int typeId() { return 5; }
    }

    public record DoubleTag(double value) implements Tag {
        @Override public int typeId() { return 6; }
    }

    public static final class ByteArrayTag implements Tag {
        private final byte[] value;
        public ByteArrayTag(byte[] value) { this.value = value.clone(); }
        public byte[] value() { return value.clone(); }
        byte[] rawValue() { return value; }
        @Override public int typeId() { return 7; }
    }

    public record StringTag(String value) implements Tag {
        public StringTag {
            if (value == null) throw new NullPointerException("NBT string");
        }
        @Override public int typeId() { return 8; }
    }

    public static final class ListTag implements Tag {
        private final int elementType;
        private final List<Tag> value;

        public ListTag(List<? extends Tag> tags) {
            if (tags == null) throw new NullPointerException("NBT list");
            List<Tag> copy = new ArrayList<>(tags.size());
            int type = 0;
            for (Tag tag : tags) {
                if (tag == null) throw new NullPointerException("NBT list element");
                if (type == 0) type = tag.typeId();
                if (tag.typeId() != type) {
                    throw new IllegalArgumentException("NBT list elements must have one tag type");
                }
                copy.add(tag);
            }
            this.elementType = type;
            this.value = List.copyOf(copy);
        }

        public int elementType() { return elementType; }
        public List<Tag> value() { return value; }
        @Override public int typeId() { return 9; }
    }

    public static final class CompoundTag implements Tag {
        private final LinkedHashMap<String, Tag> value = new LinkedHashMap<>();

        public CompoundTag put(String name, Tag tag) {
            if (name == null || tag == null) {
                throw new NullPointerException("NBT compound name/tag");
            }
            value.put(name, tag);
            return this;
        }

        public Map<String, Tag> value() {
            return Map.copyOf(value);
        }

        LinkedHashMap<String, Tag> rawValue() {
            return value;
        }

        @Override public int typeId() { return 10; }
    }

    public static final class IntArrayTag implements Tag {
        private final int[] value;
        public IntArrayTag(int[] value) { this.value = value.clone(); }
        public int[] value() { return value.clone(); }
        int[] rawValue() { return value; }
        @Override public int typeId() { return 11; }
    }

    public static final class LongArrayTag implements Tag {
        private final long[] value;
        public LongArrayTag(long[] value) { this.value = value.clone(); }
        public long[] value() { return value.clone(); }
        long[] rawValue() { return value; }
        @Override public int typeId() { return 12; }
    }

    public static ByteTag byteTag(int value) {
        return new ByteTag((byte) value);
    }

    public static ShortTag shortTag(int value) {
        return new ShortTag((short) value);
    }

    public static IntTag intTag(int value) {
        return new IntTag(value);
    }

    public static LongTag longTag(long value) {
        return new LongTag(value);
    }

    public static FloatTag floatTag(float value) {
        return new FloatTag(value);
    }

    public static DoubleTag doubleTag(double value) {
        return new DoubleTag(value);
    }

    public static ByteArrayTag byteArray(byte[] value) {
        return new ByteArrayTag(value);
    }

    public static StringTag string(String value) {
        return new StringTag(value);
    }

    public static ListTag list(List<? extends Tag> value) {
        return new ListTag(value);
    }

    public static CompoundTag compound() {
        return new CompoundTag();
    }

    public static IntArrayTag intArray(int[] value) {
        return new IntArrayTag(value);
    }

    public static LongArrayTag longArray(long[] value) {
        return new LongArrayTag(value);
    }

    public static byte[] writeRoot(String name, Tag root) {
        if (name == null || root == null) {
            throw new NullPointerException("NBT root name/tag");
        }
        try {
            ByteArrayOutputStream bytes = new ByteArrayOutputStream();
            try (DataOutputStream out = new DataOutputStream(bytes)) {
                out.writeByte(root.typeId());
                writeString(out, name);
                writePayload(out, root);
            }
            return bytes.toByteArray();
        } catch (IOException e) {
            throw new IllegalStateException("unexpected in-memory NBT I/O failure", e);
        }
    }

    private static void writePayload(DataOutputStream out, Tag tag) throws IOException {
        switch (tag.typeId()) {
            case 1 -> out.writeByte(((ByteTag) tag).value());
            case 2 -> out.writeShort(((ShortTag) tag).value());
            case 3 -> out.writeInt(((IntTag) tag).value());
            case 4 -> out.writeLong(((LongTag) tag).value());
            case 5 -> out.writeFloat(((FloatTag) tag).value());
            case 6 -> out.writeDouble(((DoubleTag) tag).value());
            case 7 -> {
                byte[] value = ((ByteArrayTag) tag).rawValue();
                out.writeInt(value.length);
                out.write(value);
            }
            case 8 -> writeString(out, ((StringTag) tag).value());
            case 9 -> {
                ListTag list = (ListTag) tag;
                out.writeByte(list.elementType());
                out.writeInt(list.value().size());
                for (Tag item : list.value()) {
                    writePayload(out, item);
                }
            }
            case 10 -> {
                CompoundTag compound = (CompoundTag) tag;
                for (Map.Entry<String, Tag> entry : compound.rawValue().entrySet()) {
                    Tag child = entry.getValue();
                    out.writeByte(child.typeId());
                    writeString(out, entry.getKey());
                    writePayload(out, child);
                }
                out.writeByte(0);
            }
            case 11 -> {
                int[] value = ((IntArrayTag) tag).rawValue();
                out.writeInt(value.length);
                for (int item : value) out.writeInt(item);
            }
            case 12 -> {
                long[] value = ((LongArrayTag) tag).rawValue();
                out.writeInt(value.length);
                for (long item : value) out.writeLong(item);
            }
            default -> throw new IllegalArgumentException("unsupported NBT tag type: " + tag.typeId());
        }
    }

    private static void writeString(DataOutputStream out, String value) throws IOException {
        byte[] bytes = value.getBytes(StandardCharsets.UTF_8);
        if (bytes.length > 0xffff) {
            throw new IllegalArgumentException("NBT string exceeds unsigned-short UTF-8 length");
        }
        out.writeShort(bytes.length);
        out.write(bytes);
    }
}
