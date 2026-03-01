// YUV420 → RGB WebGL renderer for DVR decoded video frames

const VERTEX_SHADER = `
  attribute vec4 aPosition;
  attribute vec2 aTexCoord;
  varying vec2 vTexCoord;
  void main() {
    gl_Position = aPosition;
    vTexCoord = aTexCoord;
  }
`;

const FRAGMENT_SHADER = `
  precision mediump float;
  varying vec2 vTexCoord;
  uniform sampler2D yTexture;
  uniform sampler2D uTexture;
  uniform sampler2D vTexture;
  void main() {
    float y = texture2D(yTexture, vTexCoord).r;
    float u = texture2D(uTexture, vTexCoord).r - 0.5;
    float v = texture2D(vTexture, vTexCoord).r - 0.5;
    float r = y + 1.402 * v;
    float g = y - 0.344136 * u - 0.714136 * v;
    float b = y + 1.772 * u;
    gl_FragColor = vec4(r, g, b, 1.0);
  }
`;

export class WebGLRenderer {
  private gl: WebGLRenderingContext;
  private program: WebGLProgram;
  private yTex: WebGLTexture;
  private uTex: WebGLTexture;
  private vTex: WebGLTexture;

  constructor(canvas: HTMLCanvasElement) {
    const gl = canvas.getContext("webgl", { preserveDrawingBuffer: false });
    if (!gl) throw new Error("WebGL not supported");
    this.gl = gl;

    this.program = this.createProgram(VERTEX_SHADER, FRAGMENT_SHADER);
    gl.useProgram(this.program);

    // Fullscreen quad
    const vertices = new Float32Array([
      -1, -1, 0, 1, 1, -1, 1, 1, -1, 1, 0, 0, 1, 1, 1, 0,
    ]);
    const buf = gl.createBuffer()!;
    gl.bindBuffer(gl.ARRAY_BUFFER, buf);
    gl.bufferData(gl.ARRAY_BUFFER, vertices, gl.STATIC_DRAW);

    const aPos = gl.getAttribLocation(this.program, "aPosition");
    gl.enableVertexAttribArray(aPos);
    gl.vertexAttribPointer(aPos, 2, gl.FLOAT, false, 16, 0);

    const aTex = gl.getAttribLocation(this.program, "aTexCoord");
    gl.enableVertexAttribArray(aTex);
    gl.vertexAttribPointer(aTex, 2, gl.FLOAT, false, 16, 8);

    // Create textures
    this.yTex = this.createTexture(0, "yTexture");
    this.uTex = this.createTexture(1, "uTexture");
    this.vTex = this.createTexture(2, "vTexture");
  }

  renderFrame(
    yuvData: Uint8Array,
    width: number,
    height: number,
    yLen: number,
    uvLen: number,
  ): void {
    const gl = this.gl;
    const canvas = gl.canvas as HTMLCanvasElement;

    if (canvas.width !== width || canvas.height !== height) {
      canvas.width = width;
      canvas.height = height;
    }
    gl.viewport(0, 0, width, height);

    const yData = yuvData.subarray(0, yLen);
    const uData = yuvData.subarray(yLen, yLen + uvLen);
    const vData = yuvData.subarray(yLen + uvLen, yLen + 2 * uvLen);

    this.updateTexture(this.yTex, 0, width, height, yData);
    this.updateTexture(this.uTex, 1, width / 2, height / 2, uData);
    this.updateTexture(this.vTex, 2, width / 2, height / 2, vData);

    gl.drawArrays(gl.TRIANGLE_STRIP, 0, 4);
  }

  clear(): void {
    const gl = this.gl;
    gl.clearColor(0, 0, 0, 1);
    gl.clear(gl.COLOR_BUFFER_BIT);
  }

  destroy(): void {
    const gl = this.gl;
    gl.deleteTexture(this.yTex);
    gl.deleteTexture(this.uTex);
    gl.deleteTexture(this.vTex);
    gl.deleteProgram(this.program);
  }

  private createProgram(vsSrc: string, fsSrc: string): WebGLProgram {
    const gl = this.gl;
    const vs = gl.createShader(gl.VERTEX_SHADER)!;
    gl.shaderSource(vs, vsSrc);
    gl.compileShader(vs);

    const fs = gl.createShader(gl.FRAGMENT_SHADER)!;
    gl.shaderSource(fs, fsSrc);
    gl.compileShader(fs);

    const program = gl.createProgram()!;
    gl.attachShader(program, vs);
    gl.attachShader(program, fs);
    gl.linkProgram(program);
    return program;
  }

  private createTexture(unit: number, uniformName: string): WebGLTexture {
    const gl = this.gl;
    const tex = gl.createTexture()!;
    gl.activeTexture(gl.TEXTURE0 + unit);
    gl.bindTexture(gl.TEXTURE_2D, tex);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE);
    gl.uniform1i(gl.getUniformLocation(this.program, uniformName), unit);
    return tex;
  }

  private updateTexture(
    tex: WebGLTexture,
    unit: number,
    width: number,
    height: number,
    data: Uint8Array,
  ): void {
    const gl = this.gl;
    gl.activeTexture(gl.TEXTURE0 + unit);
    gl.bindTexture(gl.TEXTURE_2D, tex);
    gl.texImage2D(
      gl.TEXTURE_2D,
      0,
      gl.LUMINANCE,
      width,
      height,
      0,
      gl.LUMINANCE,
      gl.UNSIGNED_BYTE,
      data,
    );
  }
}
