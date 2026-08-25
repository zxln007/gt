FROM nginx:alpine

# Copy static assets directly into the default nginx document root
COPY index.html /usr/share/nginx/html/index.html

# Create downloads directory within the nginx root to serve self-hosted binaries
RUN mkdir -p /usr/share/nginx/html/downloads

# Expose HTTP port 80
EXPOSE 80

# Command to run Nginx in the foreground
CMD ["nginx", "-g", "daemon off;"]
