package cn.auraoa.documentparser.parser;

import cn.auraoa.documentparser.config.ParserProperties;
import cn.auraoa.documentparser.exception.DocumentLimitExceededException;
import cn.auraoa.documentparser.exception.DocumentParserException;
import org.springframework.stereotype.Component;

import java.io.IOException;
import java.io.InputStream;
import java.nio.file.Path;
import java.util.ArrayDeque;
import java.util.Deque;
import java.util.HashSet;
import java.util.Set;
import java.util.zip.ZipEntry;
import java.util.zip.ZipFile;

/**
 * 在 OFDRW 打开文件前校验 OFD ZIP 容器，阻断路径穿越和解压炸弹。
 */
@Component
public class OfdArchiveValidator {

    private static final long RATIO_CHECK_MINIMUM_BYTES = 1024L * 1024L;
    private final ParserProperties properties;

    public OfdArchiveValidator(ParserProperties properties) {
        this.properties = properties;
    }

    public void validate(Path input) {
        long maximumUncompressedBytes = properties.maxOfdUncompressedSize().toBytes();
        long totalUncompressedBytes = 0;
        int entryCount = 0;
        boolean hasRootDocument = false;
        Set<String> names = new HashSet<>();

        try (ZipFile archive = new ZipFile(input.toFile())) {
            var entries = archive.entries();
            while (entries.hasMoreElements()) {
                ZipEntry entry = entries.nextElement();
                entryCount++;
                if (entryCount > properties.maxOfdEntries()) {
                    throw new DocumentLimitExceededException("OFD 文件条目数量超过安全上限");
                }

                String safeName = validateEntryName(entry.getName());
                if (!names.add(safeName)) {
                    throw new DocumentParserException("OFD 文件包含重复条目");
                }
                if ("OFD.xml".equals(safeName)) {
                    hasRootDocument = true;
                }
                if (entry.isDirectory()) {
                    continue;
                }

                long declaredSize = entry.getSize();
                long compressedSize = entry.getCompressedSize();
                if (declaredSize < 0 || compressedSize < 0) {
                    throw new DocumentParserException("OFD 文件条目大小信息无效");
                }
                checkCompressionRatio(declaredSize, compressedSize);

                long actualSize = countEntryBytes(archive, entry, maximumUncompressedBytes - totalUncompressedBytes);
                totalUncompressedBytes = safeAdd(totalUncompressedBytes, actualSize);
                if (totalUncompressedBytes > maximumUncompressedBytes) {
                    throw new DocumentLimitExceededException("OFD 文件解压后大小超过安全上限");
                }
                checkCompressionRatio(actualSize, compressedSize);
            }
        } catch (DocumentLimitExceededException | DocumentParserException exception) {
            throw exception;
        } catch (IOException exception) {
            throw new DocumentParserException("OFD 文件容器无效", exception);
        }

        if (!hasRootDocument) {
            throw new DocumentParserException("OFD 文件缺少 OFD.xml");
        }
    }

    private String validateEntryName(String originalName) {
        if (originalName == null || originalName.isBlank() || originalName.length() > 1024) {
            throw new DocumentParserException("OFD 文件包含无效条目名称");
        }
        String name = originalName.replace('\\', '/');
        if (name.startsWith("/") || name.indexOf('\0') >= 0 || name.matches("^[A-Za-z]:.*")) {
            throw new DocumentParserException("OFD 文件包含不安全条目路径");
        }
        Deque<String> segments = new ArrayDeque<>();
        for (String segment : name.split("/")) {
            if (segment.isEmpty() || ".".equals(segment)) {
                continue;
            }
            if ("..".equals(segment)) {
                if (segments.isEmpty()) {
                    throw new DocumentParserException("OFD 文件包含不安全条目路径");
                }
                segments.removeLast();
            } else {
                segments.addLast(segment);
            }
        }
        if (segments.isEmpty()) {
            throw new DocumentParserException("OFD 文件包含无效条目名称");
        }
        return String.join("/", segments);
    }

    private long countEntryBytes(ZipFile archive, ZipEntry entry, long remainingLimit) throws IOException {
        if (remainingLimit < 0) {
            throw new DocumentLimitExceededException("OFD 文件解压后大小超过安全上限");
        }
        long count = 0;
        byte[] buffer = new byte[16 * 1024];
        try (InputStream stream = archive.getInputStream(entry)) {
            int read;
            while ((read = stream.read(buffer)) != -1) {
                count = safeAdd(count, read);
                if (count > remainingLimit) {
                    throw new DocumentLimitExceededException("OFD 文件解压后大小超过安全上限");
                }
            }
        }
        return count;
    }

    private void checkCompressionRatio(long uncompressedSize, long compressedSize) {
        if (uncompressedSize < RATIO_CHECK_MINIMUM_BYTES) {
            return;
        }
        if (compressedSize == 0
                || (double) uncompressedSize / (double) compressedSize > properties.maxOfdCompressionRatio()) {
            throw new DocumentLimitExceededException("OFD 文件压缩比超过安全上限");
        }
    }

    private long safeAdd(long left, long right) {
        try {
            return Math.addExact(left, right);
        } catch (ArithmeticException exception) {
            throw new DocumentLimitExceededException("OFD 文件大小超过安全上限");
        }
    }
}
