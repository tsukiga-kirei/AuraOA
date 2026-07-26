package cn.auraoa.documentparser.util;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.Comparator;

/**
 * 单次请求专用临时目录，关闭时递归清理。
 */
public final class TempWorkspace implements AutoCloseable {

    private static final Logger LOGGER = LoggerFactory.getLogger(TempWorkspace.class);
    private static final String PREFIX = "auraoa-document-parser-";
    private final Path directory;

    private TempWorkspace(Path directory) {
        this.directory = directory;
    }

    public static TempWorkspace create() {
        try {
            return new TempWorkspace(Files.createTempDirectory(PREFIX).toAbsolutePath().normalize());
        } catch (IOException exception) {
            throw new IllegalStateException("无法创建文档解析临时目录", exception);
        }
    }

    public Path resolve(String fixedFilename) {
        Path candidate = directory.resolve(fixedFilename).normalize();
        if (!candidate.getParent().equals(directory)) {
            throw new IllegalArgumentException("临时文件名无效");
        }
        return candidate;
    }

    Path directory() {
        return directory;
    }

    @Override
    public void close() {
        if (!directory.getFileName().toString().startsWith(PREFIX) || !Files.exists(directory)) {
            return;
        }
        try (var paths = Files.walk(directory)) {
            paths.sorted(Comparator.reverseOrder()).forEach(this::deleteQuietly);
        } catch (IOException exception) {
            LOGGER.warn("清理文档解析临时目录失败，错误类型={}", exception.getClass().getSimpleName());
        }
    }

    private void deleteQuietly(Path path) {
        try {
            Files.deleteIfExists(path);
        } catch (IOException exception) {
            LOGGER.warn("清理文档解析临时文件失败，错误类型={}", exception.getClass().getSimpleName());
        }
    }
}
