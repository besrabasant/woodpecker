import WoodpeckerClient from '~/lib/api';
import { getStoredAuthToken } from '~/lib/authToken';

import useConfig from './useConfig';

let apiClient: WoodpeckerClient | undefined;

export default (): WoodpeckerClient => {
  if (!apiClient) {
    const config = useConfig();
    const server = config.rootPath;
    const token = getStoredAuthToken();
    const csrf = config.csrf ?? null;

    apiClient = new WoodpeckerClient(server, token, csrf);
  }

  return apiClient;
};
