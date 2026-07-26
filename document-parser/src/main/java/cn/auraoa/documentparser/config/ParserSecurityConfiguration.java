package cn.auraoa.documentparser.config;

import jakarta.annotation.PostConstruct;
import org.apache.poi.util.IOUtils;
import org.springframework.context.annotation.Configuration;

/**
 * 初始化第三方解析库的全局内存保护参数。
 */
@Configuration
public class ParserSecurityConfiguration {

    private final ParserProperties properties;

    public ParserSecurityConfiguration(ParserProperties properties) {
        this.properties = properties;
    }

    @PostConstruct
    void configurePoiLimits() {
        long limit = properties.maxPoiAllocationSize().toBytes();
        if (limit > Integer.MAX_VALUE) {
            limit = Integer.MAX_VALUE;
        }
        IOUtils.setByteArrayMaxOverride((int) limit);
    }
}
