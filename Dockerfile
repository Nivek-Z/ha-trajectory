FROM scratch
COPY ha-trajectory /ha-trajectory
EXPOSE 8080
ENTRYPOINT ["/ha-trajectory"]
