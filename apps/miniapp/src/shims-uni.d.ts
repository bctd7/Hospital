export {};

declare module "vue" {
  type UniHooks = App.AppInstance & Page.PageInstance;

  interface ComponentCustomOptions extends UniHooks {}
}
