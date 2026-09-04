FROM nginx:alpine

# Nginx site config: markdown docs MIME + revalidation
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Copy static assets directly into the default nginx document root
COPY index.html /usr/share/nginx/html/index.html
COPY install.sh /usr/share/nginx/html/install.sh
COPY assets/ /usr/share/nginx/html/assets/
COPY zh/ /usr/share/nginx/html/zh/
COPY docs/ /usr/share/nginx/html/docs/
COPY terms/ /usr/share/nginx/html/terms/
COPY abuse/ /usr/share/nginx/html/abuse/
COPY desktop-version.json /usr/share/nginx/html/desktop-version.json

# Create downloads directory within the nginx root to serve self-hosted binaries
RUN mkdir -p /usr/share/nginx/html/downloads

# Expose HTTP port 80
EXPOSE 80

# Command to run Nginx in the foreground
CMD ["nginx", "-g", "daemon off;"]
