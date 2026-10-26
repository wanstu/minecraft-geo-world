package geoworld.minecraft.common.worldgen;

/**
 * Version-neutral shoreline shaping rules. These operate only on integer
 * terrain/water boundaries and are reusable by every supported MC version.
 */
public final class ShorelineRules {
    public static final int MAX_SHORE_DISTANCE = 4;
    public static final int MAX_GENTLE_BANK_DISTANCE = 14;
    // v1.1.10 adaptive banks do not end at this distance. The value is only
    // a safety/search bound large enough for the supported low/medium relief;
    // each bank stops as soon as its constructed slope is within one block
    // of the untouched terrain.
    public static final int MAX_ADAPTIVE_BANK_DISTANCE = 32;

    private ShorelineRules() {}

    public static int gentleBankCap(
        int originalGround,
        int waterBoundary,
        int distance
    ) {
        if (distance >= MAX_GENTLE_BANK_DISTANCE) return originalGround;
        double t = Math.max(
            0.0,
            distance / (double) MAX_GENTLE_BANK_DISTANCE
        );
        double ease = t * t * (3.0 - 2.0 * t);
        return waterBoundary + Math.max(
            1,
            (int) Math.round((originalGround - waterBoundary) * ease)
        );
    }

    public static int adaptiveGentleBankRise(int distance) {
        // Two vertical blocks per five horizontal blocks. Integer rounding
        // produces a natural 0/1 staircase: typically one rise every 2-3
        // inland columns instead of a forced one-block rise on every column.
        if (distance <= 0) return 0;
        return Math.max(1, (2 * distance + 4) / 5);
    }

    public static int adaptiveGentleBankDrop(int distance) {
        // Keep the immediate dry rim level with the water, then descend with
        // the same 2-in-5 cadence when reconnecting to lower natural ground.
        if (distance <= 1) return 0;
        return (2 * (distance - 1) + 4) / 5;
    }

    public static int adaptiveGentleBankMaxRelief() {
        return adaptiveGentleBankRise(MAX_ADAPTIVE_BANK_DISTANCE);
    }

    public static int adaptiveGentleBankCap(
        int originalGround,
        int waterBoundary,
        int distance
    ) {
        int cap = waterBoundary + adaptiveGentleBankRise(distance);
        return Math.min(originalGround, cap);
    }

    public static int adaptiveGentleBankFloor(
        int originalGround,
        int waterBoundary,
        int distance
    ) {
        int floor = waterBoundary - adaptiveGentleBankDrop(distance);
        return Math.max(originalGround, floor);
    }

    public static int oceanBackshoreMinimum(
        int shoreDistance,
        int waterBoundary,
        int originalGround
    ) {
        int legacyMinimum = switch (shoreDistance) {
            case 1, 2 -> waterBoundary;
            case 3 -> waterBoundary - 1;
            case 4 -> waterBoundary - 2;
            default -> Integer.MIN_VALUE;
        };
        int depression = waterBoundary - originalGround;
        if (
            depression < 1 ||
            depression > 4 ||
            shoreDistance > MAX_GENTLE_BANK_DISTANCE ||
            shoreDistance <= 0
        ) {
            return legacyMinimum;
        }
        int taperDistance = Math.min(
            MAX_GENTLE_BANK_DISTANCE,
            8 + depression * 2
        );
        double progress = Math.min(
            1.0,
            Math.max(
                0.0,
                (shoreDistance - 2.0) / (taperDistance - 2.0)
            )
        );
        int rampFloor = waterBoundary -
            (int) Math.round(depression * progress);
        if (shoreDistance > 2) {
            rampFloor = Math.min(rampFloor, originalGround + 3);
        }
        return Math.max(legacyMinimum, rampFloor);
    }
}
