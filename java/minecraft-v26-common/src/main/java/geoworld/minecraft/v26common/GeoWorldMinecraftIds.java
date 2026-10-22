package geoworld.minecraft.v26common;

import net.minecraft.resources.ResourceKey;

import java.lang.reflect.Method;
import java.lang.reflect.Modifier;
import java.util.Optional;

/**
 * Small compatibility bridge for the resource-id rename between Minecraft
 * 1.21.x (ResourceLocation) and 26.x (Identifier).
 *
 * The world-generation implementation stays shared; only identifier creation
 * and ResourceKey id access cross this reflective boundary.
 */
public final class GeoWorldMinecraftIds {
    private static final Class<?> ID_CLASS = loadIdClass();

    private GeoWorldMinecraftIds() {}

    public static Object id(String value) {
        String namespace = "minecraft";
        String path = value;
        int colon = value.indexOf(':');
        if (colon >= 0) {
            namespace = value.substring(0, colon);
            path = value.substring(colon + 1);
        }
        return id(namespace, path);
    }

    public static Object id(String namespace, String path) {
        for (String methodName : new String[] {
            "fromNamespaceAndPath",
            "tryBuild"
        }) {
            try {
                Method method = ID_CLASS.getMethod(
                    methodName,
                    String.class,
                    String.class
                );
                Object value = method.invoke(null, namespace, path);
                if (value != null) {
                    return value;
                }
            } catch (ReflectiveOperationException ignored) {
            }
        }

        try {
            Method parse = ID_CLASS.getMethod("parse", String.class);
            return parse.invoke(null, namespace + ":" + path);
        } catch (ReflectiveOperationException exception) {
            throw new IllegalStateException(
                "Unable to create Minecraft resource id " +
                namespace + ":" + path,
                exception
            );
        }
    }

    @SuppressWarnings("unchecked")
    public static <T> ResourceKey<T> resourceKey(
        ResourceKey<?> registryKey,
        String id
    ) {
        Object location = id(id);
        for (Method method : ResourceKey.class.getMethods()) {
            if (!method.getName().equals("create") ||
                !Modifier.isStatic(method.getModifiers()) ||
                method.getParameterCount() != 2) {
                continue;
            }
            Class<?>[] parameters = method.getParameterTypes();
            if (parameters[0].isInstance(registryKey) &&
                parameters[1].isInstance(location)) {
                try {
                    return (ResourceKey<T>)method.invoke(
                        null,
                        registryKey,
                        location
                    );
                } catch (ReflectiveOperationException exception) {
                    throw new IllegalStateException(
                        "Unable to create ResourceKey for " + id,
                        exception
                    );
                }
            }
        }
        throw new IllegalStateException(
            "Compatible ResourceKey.create method not found for " +
            ID_CLASS.getName()
        );
    }

    public static String resourceKeyId(ResourceKey<?> key) {
        for (String methodName : new String[] {"identifier", "location"}) {
            try {
                Method method = key.getClass().getMethod(methodName);
                Object value = method.invoke(key);
                if (value != null) {
                    return value.toString();
                }
            } catch (ReflectiveOperationException ignored) {
            }
        }
        throw new IllegalStateException(
            "Unable to read resource id from ResourceKey " + key
        );
    }

    public static Object registryValue(Object registry, String id) {
        Object location = id(id);
        for (String methodName : new String[] {"getValue", "get"}) {
            for (Method method : registry.getClass().getMethods()) {
                if (!method.getName().equals(methodName) ||
                    method.getParameterCount() != 1 ||
                    !method.getParameterTypes()[0].isInstance(location)) {
                    continue;
                }
                try {
                    Object value = method.invoke(registry, location);
                    if (value instanceof Optional<?> optional) {
                        return optional.orElse(null);
                    }
                    return value;
                } catch (ReflectiveOperationException exception) {
                    throw new IllegalStateException(
                        "Unable to resolve registry value " + id,
                        exception
                    );
                }
            }
        }
        throw new IllegalStateException(
            "Compatible registry lookup method not found for " +
            ID_CLASS.getName()
        );
    }

    public static String namespace(String id) {
        int colon = id.indexOf(':');
        return colon < 0 ? "minecraft" : id.substring(0, colon);
    }

    public static String path(String id) {
        int colon = id.indexOf(':');
        return colon < 0 ? id : id.substring(colon + 1);
    }

    private static Class<?> loadIdClass() {
        for (String name : new String[] {
            "net.minecraft.resources.Identifier",
            "net.minecraft.resources.ResourceLocation"
        }) {
            try {
                return Class.forName(name);
            } catch (ClassNotFoundException ignored) {
            }
        }
        throw new ExceptionInInitializerError(
            "No supported Minecraft resource id class is available"
        );
    }
}
