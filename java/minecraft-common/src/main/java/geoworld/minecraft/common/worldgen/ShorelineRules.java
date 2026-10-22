package geoworld.minecraft.common.worldgen;

/**
 * Version-neutral shoreline shaping rules. These operate only on integer
 * terrain/water boundaries and are reusable by every supported MC version.
 */
public final class ShorelineRules {
    public static final int MAX_SHORE_DISTANCE = 4;
    public static final int MAX_GENTLE_BANK_DISTANCE = 14;

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
