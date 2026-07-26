package cn.auraoa.documentparser.parser;

import cn.auraoa.documentparser.exception.DocumentLimitExceededException;

import java.io.FilterOutputStream;
import java.io.IOException;
import java.io.OutputStream;

/**
 * 防止第三方导出器向内存写入无限数据。
 */
final class LimitedOutputStream extends FilterOutputStream {

    private final long limit;
    private long written;

    LimitedOutputStream(OutputStream output, long limit) {
        super(output);
        this.limit = limit;
    }

    @Override
    public void write(int value) throws IOException {
        ensureCapacity(1);
        out.write(value);
        written++;
    }

    @Override
    public void write(byte[] buffer, int offset, int length) throws IOException {
        ensureCapacity(length);
        out.write(buffer, offset, length);
        written += length;
    }

    private void ensureCapacity(int additionalBytes) {
        if (additionalBytes < 0 || written > limit - additionalBytes) {
            throw new DocumentLimitExceededException("解析输出超过安全长度上限");
        }
    }
}
