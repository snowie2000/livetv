# Step 1: Use Python 3.14 based on Debian 13 (Trixie)
FROM python:3.14-slim-trixie

# Step 2: Set environment variables
ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1

# Step 3: Install system dependencies
# Added xz-utils which is sometimes needed for node/npm extracts
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    git \
    ca-certificates \
    procps \
    unzip \
    xz-utils \
    && rm -rf /var/lib/apt/lists/*

# Step 4: Install Node.js LTS (Official NodeSource Method)
# This installs both 'node' and 'npm' correctly
RUN curl -fsSL https://deb.nodesource.com/setup_lts.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*

# Step 5: Install Deno (Latest official binary)
COPY --from=denoland/deno:bin /deno /usr/local/bin/deno

# Step 6: Install yt-dlp globally
# Using --break-system-packages as required by Debian 13/Python 3.14
RUN pip install --no-cache-dir yt-dlp[default] --break-system-packages

# Step 7: Setup bgutil-ytdlp-pot-provider
WORKDIR /opt
RUN git clone --single-branch --branch 1.2.2 https://github.com/Brainicism/bgutil-ytdlp-pot-provider.git
WORKDIR /opt/bgutil-ytdlp-pot-provider/server
RUN npm install && npx tsc

# Step 8: Setup livetv
WORKDIR /opt/livetv
RUN curl -L -o livetv-linux-amd64 https://github.com/snowie2000/livetv/releases/latest/download/livetv-linux-amd64 \
    && chmod +x livetv-linux-amd64 \
    && mkdir -p /opt/livetv/data

# Step 9: Persistence (livetv data)
VOLUME ["/opt/livetv/data"]

# Step 10: Create entrypoint script for concurrent execution
RUN printf "#!/bin/bash\n\
echo \"[1/2] Starting bgutil-ytdlp-pot-provider on :4416...\"\n\
cd /opt/bgutil-ytdlp-pot-provider/server && node build/main.js &\n\
\n\
echo \"[2/2] Starting livetv on :9000...\"\n\
cd /opt/livetv && ./livetv-linux-amd64\n\
\n\
# Wait logic ensures container stays up and cleans up on exit\n\
wait -n\n\
exit \$?" > /entrypoint.sh && chmod +x /entrypoint.sh

# Step 11: Final configuration
# 4416: bgutil-provider | 9000: livetv
EXPOSE 9000

ENTRYPOINT ["/entrypoint.sh"]