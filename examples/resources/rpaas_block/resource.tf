resource "rpaas_block" "example_http" {
  service_name = "rpaasv2-be"
  instance     = "my-rpaas"

  name    = "http" # One of [root, http, server, lua-server, lua-worker]
  content = <<-EOF
    upstream my_upstream {
      root         /usr/share/nginx/html;
      include      mime.types;
      default_type application/json;

      server_tokens      off;
      more_clear_headers Server;
    }
  EOF
}

resource "rpaas_block" "example_server" {
  service_name = "rpaasv2-be"
  instance     = "my-rpaas"

  name    = "server"
  content = <<-EOF
    default_type text/html;
    more_set_headers 'X-Frame-Options: deny';

    location / {
      if ($scheme = 'http') {
        return 301 https://${http_host}${request_uri};
      }

      proxy_set_header Connection        '';
      proxy_set_header Host              my-service.cluster.svc.cluster.local:8080;
      proxy_set_header X-Forwarded-For   ${proxy_add_x_forwarded_for};
      proxy_set_header X-Forwarded-Host  ${host};
      proxy_set_header X-Forwarded-Proto ${scheme};
      proxy_set_header X-Real-IP         ${remote_addr};
      proxy_set_header X-Request-Id      ${request_id_final};
      proxy_set_header Early-Data        ${ssl_early_data};

      proxy_pass     http://my_upstream/;
      proxy_redirect ~^http://my-service.cluster.svc.cluster.local:8080/(.*)$ /$2;
    }
  EOF
}

resource "rpaas_block" "example_lua" {
  service_name = "rpaasv2-be"
  instance     = "my-rpaas"

  name    = "lua-server"
  content = file("script.lua")
}
