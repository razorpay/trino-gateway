window.onload = function() {
  //<editor-fold desc="Changeable Configuration Block">

  // the following lines will be replaced by docker/configurator, when it runs in a docker-container
  window.ui = SwaggerUIBundle({
    url: "./rpc/gateway/service.swagger.json",
    dom_id: '#swagger-ui',
    deepLinking: true,
    presets: [
      SwaggerUIBundle.presets.apis,
      SwaggerUIStandalonePreset
    ],
    plugins: [
      SwaggerUIBundle.plugins.DownloadUrl,
      HideInfoUrlPartsPlugin,
      HideOperationsUntilAuthorizedPlugin
    ],
    layout: "StandaloneLayout"
  });

  //</editor-fold>
};

const HideInfoUrlPartsPlugin = () => {
  return {
    wrapComponents: {
      InfoUrl: () => () => null,
      // InfoBasePath: () => () => null, // this hides the `Base Url` part too, if you want that
    }
  }
}

const HideOperationsUntilAuthorizedPlugin = function() {
  return {
    wrapComponents: {
      operation: (Ori, system) => (props) => {
        const isOperationSecured = !!props.operation.get("security").size
        const isOperationAuthorized = props.operation.get("isAuthorized")

        if(!isOperationSecured || isOperationAuthorized) {
          return system.React.createElement(Ori, props)
        }
        return null
      }
    }
  }
}
