package geoworld.minecraft.v26_3;

import java.util.Arrays;

final class PackedValues {
    private PackedValues() {}

    static long[] pack(int[] values, int bits) {
        if (values == null) throw new NullPointerException("values");
        if (bits < 1 || bits > 32) {
            throw new IllegalArgumentException("bits must be in 1..32");
        }

        int valuesPerLong = 64 / bits;
        int longs = (values.length + valuesPerLong - 1) / valuesPerLong;
        long[] out = new long[longs];
        long mask = bits == 64 ? -1L : (1L << bits) - 1L;

        for (int index = 0; index < values.length; index++) {
            int value = values[index];
            if (value < 0 || (value & ~mask) != 0) {
                throw new IllegalArgumentException(
                    "value " + value + " does not fit in " + bits + " bits"
                );
            }
            int cell = index / valuesPerLong;
            int shift = (index - cell * valuesPerLong) * bits;
            out[cell] |= ((long) value & mask) << shift;
        }
        return out;
    }

    static int[] unpack(long[] data, int size, int bits) {
        if (data == null) throw new NullPointerException("data");
        int valuesPerLong = 64 / bits;
        int expected = (size + valuesPerLong - 1) / valuesPerLong;
        if (data.length != expected) {
            throw new IllegalArgumentException(
                "packed length=" + data.length + " want=" + expected
            );
        }
        long mask = (1L << bits) - 1L;
        int[] out = new int[size];
        for (int index = 0; index < size; index++) {
            int cell = index / valuesPerLong;
            int shift = (index - cell * valuesPerLong) * bits;
            out[index] = (int) ((data[cell] >>> shift) & mask);
        }
        return out;
    }

    static int ceilLog2(int values) {
        if (values <= 1) return 0;
        return 32 - Integer.numberOfLeadingZeros(values - 1);
    }

    static int blockStateBits(int paletteSize) {
        if (paletteSize <= 1) return 0;
        int required = ceilLog2(paletteSize);
        if (required <= 4) return 4;
        if (required <= 8) return required;
        throw new IllegalArgumentException(
            "preview writer does not support block-state palette size > 256"
        );
    }

    static int biomeBits(int paletteSize) {
        if (paletteSize <= 1) return 0;
        int required = ceilLog2(paletteSize);
        if (required <= 3) return required;
        throw new IllegalArgumentException(
            "preview writer keeps biome palettes local and requires <= 8 entries"
        );
    }

    static void assertRoundTrip(int[] values, int bits) {
        if (!Arrays.equals(values, unpack(pack(values, bits), values.length, bits))) {
            throw new AssertionError("packed value round-trip mismatch");
        }
    }
}
