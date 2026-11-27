FROM nginx:1.31.8-alpine AS modules

ARG HEADERS_MORE_VERSION=0.41

RUN apk add --no-cache build-base pcre2-dev zlib-dev openssl-dev linux-headers curl

RUN curl -fsSL "https://nginx.org/download/nginx-${NGINX_VERSION}.tar.gz" | tar -xz -C /tmp \
    && curl -fsSL "https://github.com/openresty/headers-more-nginx-module/archive/refs/tags/v${HEADERS_MORE_VERSION}.tar.gz" | tar -xz -C /tmp \
    && cd "/tmp/nginx-${NGINX_VERSION}" \
    && ./configure --with-compat --add-dynamic-module="/tmp/headers-more-nginx-module-${HEADERS_MORE_VERSION}" \
    && make modules \
    && mkdir /out \
    && cp objs/ngx_http_headers_more_filter_module.so /out/

FROM nginx:1.31.8-alpine

COPY --from=modules /out/ngx_http_headers_more_filter_module.so /usr/lib/nginx/modules/

RUN sed -i '1i load_module modules/ngx_http_headers_more_filter_module.so;' /etc/nginx/nginx.conf
