import { DefaultApi, Configuration } from '@generated/index';

const configuration = new Configuration({
  basePath: '/api/v1',
});

export const apiClient = new DefaultApi(configuration);
