# Dockerfile for Arch Linux with latest GCC/G++ and time command

# Use the official Arch Linux base image
FROM archlinux:latest

RUN pacman -Syu --noconfirm && \
    pacman -S --noconfirm base-devel time && \
    rm -rf /var/cache/pacman/pkg/*

RUN echo "Verifying installations..." && \
    gcc --version && \
    g++ --version && \
    /usr/bin/time --version && \
    make --version

CMD ["bash"]