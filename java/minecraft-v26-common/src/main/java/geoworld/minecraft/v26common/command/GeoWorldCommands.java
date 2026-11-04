package geoworld.minecraft.v26common.command;

import com.mojang.brigadier.CommandDispatcher;
import geoworld.minecraft.common.performance.GeoWorldChunkPerformance;
import geoworld.minecraft.v26common.runtime.GeoWorldV26WorldService;
import geoworld.runtime.GeoWorldV1;
import net.minecraft.commands.CommandSourceStack;
import net.minecraft.commands.Commands;
import net.minecraft.network.chat.Component;
import net.minecraft.server.MinecraftServer;

import java.util.Optional;

public final class GeoWorldCommands {
    @FunctionalInterface
    public interface ActiveWorldLookup {
        Optional<GeoWorldV26WorldService.ActiveWorld> active(
            MinecraftServer server
        );
    }

    private GeoWorldCommands() {}

    public static void register(
        CommandDispatcher<CommandSourceStack> dispatcher,
        ActiveWorldLookup worlds
    ) {
        dispatcher.register(
            Commands.literal("geoworld")
                .executes(context -> showInfo(context.getSource(), worlds))
                .then(
                    Commands.literal("webui")
                        .executes(context -> showWebUi(context.getSource()))
                )
                .then(
                    Commands.literal("perf")
                        .executes(context -> showPerformance(context.getSource()))
                        .then(
                            Commands.literal("on")
                                .executes(context -> setPerformance(
                                    context.getSource(),
                                    true
                                ))
                        )
                        .then(
                            Commands.literal("off")
                                .executes(context -> setPerformance(
                                    context.getSource(),
                                    false
                                ))
                        )
                        .then(
                            Commands.literal("reset")
                                .executes(context -> resetPerformance(
                                    context.getSource()
                                ))
                        )
                )
        );
    }

    private static int showInfo(
        CommandSourceStack source,
        ActiveWorldLookup worlds
    ) {
        send(source, "GeoWorld");
        Optional<GeoWorldV26WorldService.ActiveWorld> active =
            worlds.active(source.getServer());
        if (active.isEmpty()) {
            send(source, "World: not a bound GeoWorld world");
            sendWebUi(source);
            return 1;
        }

        GeoWorldV26WorldService.ActiveWorld world = active.get();
        GeoWorldV1.Manifest manifest = world.model().manifest();
        send(
            source,
            "World: active, dynamicWorldgen=" + world.dynamicWorldgen()
        );
        send(
            source,
            "Minecraft=" + world.binding().minecraftVersion() +
            ", seed=" + manifest.seed()
        );
        send(
            source,
            "Generator=" + manifest.generatorVersion() +
            ", rules=" + manifest.generationRulesVersion()
        );
        send(
            source,
            "Model=" + manifest.worldModelVersion() +
            ", profile=" + manifest.worldProfileVersion()
        );
        send(source, "Model file: " + world.binding().modelPath());
        sendWebUi(source);
        return 1;
    }

    private static int showWebUi(CommandSourceStack source) {
        sendWebUi(source);
        return 1;
    }

    private static void sendWebUi(CommandSourceStack source) {
        String url = System.getProperty("geoworld.webui.url", "").trim();
        send(
            source,
            url.isEmpty()
                ? "WebUI: not running / unavailable in this process"
                : "WebUI: " + url
        );
    }

    private static int showPerformance(CommandSourceStack source) {
        send(
            source,
            "Chunk performance: " + GeoWorldChunkPerformance.summary()
        );
        return 1;
    }

    private static int setPerformance(
        CommandSourceStack source,
        boolean enabled
    ) {
        GeoWorldChunkPerformance.setEnabled(enabled);
        if (enabled) {
            GeoWorldChunkPerformance.reset();
        }
        send(
            source,
            "Chunk performance profiling " +
            (enabled ? "enabled" : "disabled")
        );
        if (enabled) {
            send(
                source,
                "Samples reset; fly into new chunks, then run /geoworld perf"
            );
        }
        return 1;
    }

    private static int resetPerformance(CommandSourceStack source) {
        GeoWorldChunkPerformance.reset();
        send(source, "Chunk performance samples reset");
        return 1;
    }

    private static void send(CommandSourceStack source, String message) {
        source.sendSuccess(() -> Component.literal(message), false);
    }
}
