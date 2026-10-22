package geoworld.minecraft.common.platform;

import java.util.function.Consumer;

/**
 * Loader-neutral lifecycle bridge.
 *
 * @param <S> the Minecraft server type exposed by one supported game version
 */
public interface ServerLifecycleAdapter<S> {
    void registerServerLifecycle(
        Consumer<S> onReady,
        Consumer<S> onStopped
    );
}
