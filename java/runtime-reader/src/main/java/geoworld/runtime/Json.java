package geoworld.runtime;

import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

final class Json {
    private Json() {}

    record Root(Map<String, Object> object, String rawPayload) {}

    static Object parse(String source) {
        Parser parser = new Parser(source);
        Object value = parser.parseValue();
        parser.skipWhitespace();
        if (!parser.isEnd()) {
            throw parser.error("trailing JSON content");
        }
        return value;
    }

    static Root parseEnvelope(String source) {
        Parser parser = new Parser(source);
        parser.skipWhitespace();
        parser.expect('{');
        Map<String, Object> object = new LinkedHashMap<>();
        String rawPayload = null;

        parser.skipWhitespace();
        if (parser.consume('}')) {
            throw parser.error("empty envelope");
        }

        while (true) {
            parser.skipWhitespace();
            String key = parser.parseString();
            parser.skipWhitespace();
            parser.expect(':');
            parser.skipWhitespace();
            int start = parser.position();
            Object value = parser.parseValue();
            int end = parser.position();
            object.put(key, value);
            if ("payload".equals(key)) {
                rawPayload = source.substring(start, end);
            }
            parser.skipWhitespace();
            if (parser.consume('}')) {
                break;
            }
            parser.expect(',');
        }

        parser.skipWhitespace();
        if (!parser.isEnd()) {
            throw parser.error("trailing JSON content");
        }
        if (rawPayload == null) {
            throw new IllegalArgumentException("GeoWorld envelope has no payload");
        }
        return new Root(object, rawPayload);
    }

    @SuppressWarnings("unchecked")
    static Map<String, Object> object(Object value, String name) {
        if (!(value instanceof Map<?, ?> map)) {
            throw new IllegalArgumentException(name + " must be a JSON object");
        }
        return (Map<String, Object>) map;
    }

    @SuppressWarnings("unchecked")
    static List<Object> array(Object value, String name) {
        if (!(value instanceof List<?> list)) {
            throw new IllegalArgumentException(name + " must be a JSON array");
        }
        return (List<Object>) list;
    }

    static String string(Map<String, Object> object, String key) {
        Object value = required(object, key);
        if (!(value instanceof String s)) {
            throw new IllegalArgumentException(key + " must be a string");
        }
        return s;
    }

    static boolean bool(Map<String, Object> object, String key) {
        Object value = required(object, key);
        if (!(value instanceof Boolean b)) {
            throw new IllegalArgumentException(key + " must be a boolean");
        }
        return b;
    }

    static long longValue(Map<String, Object> object, String key) {
        Object value = required(object, key);
        if (!(value instanceof Number n)) {
            throw new IllegalArgumentException(key + " must be a number");
        }
        return n.longValue();
    }

    static int intValue(Map<String, Object> object, String key) {
        long value = longValue(object, key);
        if (value < Integer.MIN_VALUE || value > Integer.MAX_VALUE) {
            throw new IllegalArgumentException(key + " is outside int range");
        }
        return (int) value;
    }

    static double doubleValue(Map<String, Object> object, String key) {
        Object value = required(object, key);
        if (!(value instanceof Number n)) {
            throw new IllegalArgumentException(key + " must be a number");
        }
        return n.doubleValue();
    }

    static Object required(Map<String, Object> object, String key) {
        if (!object.containsKey(key)) {
            throw new IllegalArgumentException("missing JSON field: " + key);
        }
        return object.get(key);
    }

    private static final class Parser {
        private final String source;
        private int pos;

        Parser(String source) {
            this.source = source;
        }

        int position() {
            return pos;
        }

        boolean isEnd() {
            return pos >= source.length();
        }

        IllegalArgumentException error(String message) {
            return new IllegalArgumentException(message + " at JSON offset " + pos);
        }

        void skipWhitespace() {
            while (!isEnd()) {
                char c = source.charAt(pos);
                if (c == ' ' || c == '\n' || c == '\r' || c == '\t') {
                    pos++;
                } else {
                    break;
                }
            }
        }

        boolean consume(char expected) {
            if (!isEnd() && source.charAt(pos) == expected) {
                pos++;
                return true;
            }
            return false;
        }

        void expect(char expected) {
            if (isEnd() || source.charAt(pos) != expected) {
                throw error("expected '" + expected + "'");
            }
            pos++;
        }

        Object parseValue() {
            skipWhitespace();
            if (isEnd()) {
                throw error("unexpected end of JSON");
            }
            return switch (source.charAt(pos)) {
                case '{' -> parseObject();
                case '[' -> parseArray();
                case '"' -> parseString();
                case 't' -> parseLiteral("true", Boolean.TRUE);
                case 'f' -> parseLiteral("false", Boolean.FALSE);
                case 'n' -> parseLiteral("null", null);
                default -> parseNumber();
            };
        }

        Map<String, Object> parseObject() {
            expect('{');
            Map<String, Object> object = new LinkedHashMap<>();
            skipWhitespace();
            if (consume('}')) {
                return object;
            }
            while (true) {
                skipWhitespace();
                String key = parseString();
                skipWhitespace();
                expect(':');
                Object value = parseValue();
                object.put(key, value);
                skipWhitespace();
                if (consume('}')) {
                    return object;
                }
                expect(',');
            }
        }

        List<Object> parseArray() {
            expect('[');
            List<Object> array = new ArrayList<>();
            skipWhitespace();
            if (consume(']')) {
                return array;
            }
            while (true) {
                array.add(parseValue());
                skipWhitespace();
                if (consume(']')) {
                    return array;
                }
                expect(',');
            }
        }

        String parseString() {
            expect('"');
            StringBuilder out = new StringBuilder();
            while (!isEnd()) {
                char c = source.charAt(pos++);
                if (c == '"') {
                    return out.toString();
                }
                if (c == '\\') {
                    if (isEnd()) {
                        throw error("unfinished string escape");
                    }
                    char escaped = source.charAt(pos++);
                    switch (escaped) {
                        case '"' -> out.append('"');
                        case '\\' -> out.append('\\');
                        case '/' -> out.append('/');
                        case 'b' -> out.append('\b');
                        case 'f' -> out.append('\f');
                        case 'n' -> out.append('\n');
                        case 'r' -> out.append('\r');
                        case 't' -> out.append('\t');
                        case 'u' -> out.append(parseUnicodeEscape());
                        default -> throw error("unsupported string escape");
                    }
                } else {
                    if (c < 0x20) {
                        throw error("control character in JSON string");
                    }
                    out.append(c);
                }
            }
            throw error("unterminated JSON string");
        }

        char parseUnicodeEscape() {
            if (pos + 4 > source.length()) {
                throw error("unfinished unicode escape");
            }
            int value = 0;
            for (int i = 0; i < 4; i++) {
                char c = source.charAt(pos++);
                int digit = Character.digit(c, 16);
                if (digit < 0) {
                    throw error("invalid unicode escape");
                }
                value = (value << 4) | digit;
            }
            return (char) value;
        }

        Object parseLiteral(String literal, Object value) {
            if (!source.startsWith(literal, pos)) {
                throw error("invalid JSON literal");
            }
            pos += literal.length();
            return value;
        }

        Number parseNumber() {
            int start = pos;
            if (consume('-')) {
                if (isEnd()) {
                    throw error("unfinished number");
                }
            }
            if (consume('0')) {
                // A single zero is valid; fraction/exponent may follow.
            } else {
                if (isEnd() || !Character.isDigit(source.charAt(pos))) {
                    throw error("invalid number");
                }
                while (!isEnd() && Character.isDigit(source.charAt(pos))) {
                    pos++;
                }
            }

            boolean floating = false;
            if (consume('.')) {
                floating = true;
                if (isEnd() || !Character.isDigit(source.charAt(pos))) {
                    throw error("invalid number fraction");
                }
                while (!isEnd() && Character.isDigit(source.charAt(pos))) {
                    pos++;
                }
            }
            if (!isEnd() && (source.charAt(pos) == 'e' || source.charAt(pos) == 'E')) {
                floating = true;
                pos++;
                if (!isEnd() && (source.charAt(pos) == '+' || source.charAt(pos) == '-')) {
                    pos++;
                }
                if (isEnd() || !Character.isDigit(source.charAt(pos))) {
                    throw error("invalid number exponent");
                }
                while (!isEnd() && Character.isDigit(source.charAt(pos))) {
                    pos++;
                }
            }

            String text = source.substring(start, pos);
            try {
                return floating ? Double.parseDouble(text) : Long.parseLong(text);
            } catch (NumberFormatException e) {
                throw error("invalid numeric value");
            }
        }
    }
}
