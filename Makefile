.PHONY: dev dev-backend dev-frontend build run clean download-go2rtc

# Build the frontend then compile Go binary with embedded assets
build: web/dist
	go build -o surveillance-client .

# Build frontend
web/dist: web/node_modules web/src web/index.html web/vite.config.ts
	cd web && npm run build

web/node_modules: web/package.json
	cd web && npm install
	@touch $@

# Run the built binary
run: build
	./surveillance-client

# Development: run backend and frontend concurrently
dev:
	@echo "Starting dev servers..."
	@make dev-backend & make dev-frontend & wait

dev-backend:
	@mkdir -p data web/dist
	@touch web/dist/index.html
	go run . &
	@wait

dev-frontend:
	cd web && npm install && npm run dev

# Download go2rtc binary for current platform
download-go2rtc:
	@echo "Downloading go2rtc..."
	@ARCH=$$(uname -m); OS=$$(uname -s | tr '[:upper:]' '[:lower:]'); \
	case "$$ARCH" in \
		x86_64) ARCH="amd64" ;; \
		aarch64|arm64) ARCH="arm64" ;; \
	esac; \
	case "$$OS" in \
		darwin) PLATFORM="mac" ;; \
		linux) PLATFORM="linux" ;; \
	esac; \
	if [ "$$PLATFORM" = "mac" ] || [ "$$PLATFORM" = "freebsd" ]; then \
		URL="https://github.com/AlexxIT/go2rtc/releases/latest/download/go2rtc_$${PLATFORM}_$${ARCH}.zip"; \
		echo "Downloading from $$URL"; \
		curl -L -o go2rtc.zip "$$URL" && unzip -o go2rtc.zip go2rtc && rm go2rtc.zip && chmod +x go2rtc; \
	else \
		URL="https://github.com/AlexxIT/go2rtc/releases/latest/download/go2rtc_$${PLATFORM}_$${ARCH}"; \
		echo "Downloading from $$URL"; \
		curl -L -o go2rtc "$$URL" && chmod +x go2rtc; \
	fi
	@echo "go2rtc downloaded successfully"

clean:
	rm -f surveillance-client
	rm -rf web/dist web/node_modules
	rm -f go2rtc go2rtc.yaml
	rm -rf data/
