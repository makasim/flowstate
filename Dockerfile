FROM public.ecr.aws/docker/library/alpine:3.20

WORKDIR /
ADD flowstate /flowstate

CMD ["/flowstate"]
